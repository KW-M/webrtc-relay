# Webrtc-relay examples

> Open one of the html files from the [examples/frontend](./frontend/) folder in your browser to test any of the backend examples.

The examples folder is organized by language used to write the example backend:

- **Golang:** These examples use this library directly and don't use the named pipes (exept for media streams)
- **Python:** These Python examples use the standalone command version of webrtc-relay and communicate with the relay wholey over named pipes. Any non-golang backend should be written simmilarly to the python examples.
  - To run this example, run `python3 simple.py` in a terminal at the python folder

---

- **Frontend:** The Javascript and HTML that can serve as the user interface on the other end of the relay. (In theory the "frontend" could be some non-browser peerjs implementation on another computer, but that isn't covered here)
