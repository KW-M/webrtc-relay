import asyncio, logging, os, shlex, subprocess
import errno

############################
###### setup logging #######
log = logging.getLogger(__name__)


class Named_Pipe:
    def __init__(self, pipe_file_path: str, create_pipe: bool = False):
        self.pipe_file = None
        self.pipe_file_path = pipe_file_path
        self.create_pipe = create_pipe

    async def get_open_pipe(self, mode: str, timeout: float = 1):
        print(f'PYTHON Opening pipe: {self.pipe_file_path}')
        # loop until sucessful
        while True:

            if self.create_pipe:
                # Create the pipe (if it doesn't already exist)
                while not os.path.exists(self.pipe_file_path):
                    os.makedirs(os.path.dirname(self.pipe_file_path),
                                exist_ok=True)
                    os.mkfifo(self.pipe_file_path)
            else:
                # Wait for the named pipe to be created (by some other program)
                while not os.path.exists(self.pipe_file_path):
                    print("wait" + self.pipe_file_path)
                    await asyncio.sleep(timeout)

            try:

                if mode == 'r':
                    try:
                        self.pipe_file = os.open(self.pipe_file_path,
                                                 os.O_RDWR | os.O_NONBLOCK)
                        return self.pipe_file
                    except OSError as ex:
                        if ex.errno == errno.ENXIO:
                            print(
                                "Err opening pipe file: Pipe is not yet readable."
                                + self.pipe_file_path)

                elif mode == 'w':
                    try:
                        self.pipe_file = os.open(self.pipe_file_path,
                                                 os.O_RDWR | os.O_NONBLOCK)
                        return self.pipe_file
                    except OSError as ex:
                        if ex.errno == errno.ENXIO:
                            print(
                                "Err opening pipe file: Pipe is not yet writeable."
                                + self.pipe_file_path)

                else:
                    raise ValueError('mode must be "r" or "w"')

            except Exception as e:
                log.error(f'Failed to open pipe file mode={mode}: {e}')

            await asyncio.sleep(1)

    def close(self):
        if self.pipe_file is not None:
            try:
                os.close(self.pipe_file)
            except Exception as e:
                log.warn(f'Failed to close pipe file: {e}')


class Named_Pipe_Relay:
    def __init__(self,
                 pipe_file_path: str,
                 create_pipe: bool = False,
                 max_queue_size: int = 30):
        self.pipe = Named_Pipe(pipe_file_path, create_pipe)
        # create the message queue to hold messages to send or recieve (depending on mode)
        self.pipe_message_queue = asyncio.Queue(maxsize=max_queue_size)

    async def read_loop(self, asyncLoop=None):
        if asyncLoop is None:
            asyncLoop = asyncio.get_event_loop()

        read_transport = None
        self.closed = False

        while True:
            # from: https://gist.github.com/oconnor663/08c081904264043e55bf
            try:
                pipe_file = await self.pipe.get_open_pipe('r')
                print("read pipe_file open: " + str(self.pipe.pipe_file_path))
                with os.fdopen(pipe_file, 'r') as stream:
                    reader = asyncio.StreamReader()
                    read_transport, _ = await asyncLoop.connect_read_pipe(
                        lambda: asyncio.StreamReaderProtocol(reader), stream)

                    while True:
                        data = await reader.readuntil(b'\n')
                        if data:
                            await self.pipe_message_queue.put(
                                str(data, 'utf-8').strip('\n'))

            except Exception as e:
                if read_transport != None:
                    read_transport.close()
                if self.closed == True:
                    return
                self.pipe.close()
                log.error(f'Pipe read failed: {e}')
                await asyncio.sleep(1)
                break

    async def write_loop(self, asyncLoop=None):
        if asyncLoop is None:
            asyncLoop = asyncio.get_event_loop()

        self.closed = False

        while True:
            try:
                pipe_file = await self.pipe.get_open_pipe('w')
                print("write pipe_file open: " + str(self.pipe.pipe_file_path))
                with os.fdopen(pipe_file, 'w') as stream:

                    while True:
                        msg = await self.pipe_message_queue.get()
                        if msg:
                            stream.write(msg + '\n')
                            stream.flush()

            except Exception as e:
                if self.closed == True:
                    return
                self.pipe.close()
                log.error(f'Pipe write failed: {e}')
                await asyncio.sleep(1)

    def close(self):
        self.closed = True
        self.pipe.close()


class Duplex_Named_Pipe_Relay:
    def __init__(self,
                 incoming_pipe_file_path: str,
                 outgoing_pipe_file_path: str,
                 create_pipes: bool = False,
                 max_queue_size: int = 30):
        self.max_queue_size = max_queue_size
        self.incoming_pipe = Named_Pipe_Relay(incoming_pipe_file_path,
                                              create_pipes, max_queue_size)
        self.outgoing_pipe = Named_Pipe_Relay(outgoing_pipe_file_path,
                                              create_pipes, max_queue_size)

    def is_open(self):
        return self.incoming_pipe.pipe_message_queue.qsize(
        ) < self.max_queue_size and self.outgoing_pipe.pipe_message_queue.qsize(
        ) < self.max_queue_size

    async def start_loops(self, asyncLoop=None):
        if asyncLoop is None:
            asyncLoop = asyncio.get_event_loop()

        await asyncio.gather(self.incoming_pipe.read_loop(asyncLoop),
                             self.outgoing_pipe.write_loop(asyncLoop))

    async def write_message(self, message: str):
        await self.outgoing_pipe.pipe_message_queue.put(message)

    async def get_next_message(self):
        return await self.incoming_pipe.pipe_message_queue.get()

    def cleanup(self):
        self.incoming_pipe.close()
        self.outgoing_pipe.close()


class Command_Output_To_Named_Pipe:
    def __init__(self,
                 pipe_file_path: str,
                 create_pipe: bool = False,
                 command_string: str = "",
                 input_pipe=None):
        self.command_and_args = shlex.split(command_string)
        print(self.command_and_args)
        self.input_pipe = input_pipe
        self.out_pipe = Named_Pipe(pipe_file_path, create_pipe)
        self.running_cmd = None

    async def start_cmd(self, asyncLoop=None):
        if asyncLoop is None:
            asyncLoop = asyncio.get_event_loop()

        # create or wait for the command to open
        pipe_file = await self.out_pipe.get_open_pipe('w')
        # self.out_pipe.close()
        self.running_cmd = subprocess.Popen(self.command_and_args,
                                            bufsize=0,
                                            stdin=self.input_pipe,
                                            stdout=os.fdopen(pipe_file, 'w'),
                                            stderr=None)
        return self.running_cmd

    def stop_piping_cmd(self):
        self.running_cmd.terminate()
        self.running_cmd.wait()
        self.stdout.close()
        self.running_cmd = None
