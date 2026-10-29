"""Serve the public landing page, checked-in web build, and local API."""

from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen


DIST = Path(__file__).resolve().parents[1] / "apps" / "web" / "dist"
LANDING = Path(__file__).resolve().parents[1] / "landing"
API = "http://127.0.0.1:8080"


class Handler(SimpleHTTPRequestHandler):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, directory=str(DIST), **kwargs)

    def do_GET(self):
        if self.path.startswith(("/api/", "/health", "/ready")):
            self.proxy()
        elif self.path.split("?", 1)[0] in ("/", "/login", "/landing.css", "/workspace-controls.js"):
            self.serve_landing()
        else:
            if not (DIST / self.path.lstrip("/").split("?", 1)[0]).is_file():
                self.serve_workspace()
                return
            super().do_GET()

    def serve_landing(self):
        filename = "styles.css" if self.path.startswith("/landing.css") else "workspace-controls.js" if self.path.startswith("/workspace-controls.js") else "login.html" if self.path.startswith("/login") else "index.html"
        file = LANDING / filename
        data = file.read_bytes()
        self.send_response(200)
        content_type = "text/css" if file.suffix == ".css" else "text/javascript" if file.suffix == ".js" else "text/html"
        self.send_header("Content-Type", content_type + "; charset=utf-8")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def serve_workspace(self):
        data = (DIST / "index.html").read_bytes().replace(
            b"</body>", b'<script src="/workspace-controls.js" defer></script></body>'
        )
        self.send_response(200)
        self.send_header("Content-Type", "text/html; charset=utf-8")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def do_POST(self):
        self.proxy()

    def do_PATCH(self):
        self.proxy()

    def do_DELETE(self):
        self.proxy()

    def proxy(self):
        body = self.rfile.read(int(self.headers.get("Content-Length", "0")))
        headers = {key: value for key, value in self.headers.items()
                   if key.lower() in ("authorization", "content-type", "accept")}
        request = Request(API + self.path, data=body if body else None,
                          headers=headers, method=self.command)
        try:
            response = urlopen(request, timeout=15)
        except HTTPError as error:
            response = error
        except URLError:
            self.send_error(502, "Documi API is unavailable on localhost:8080")
            return
        with response:
            self.send_response(response.status)
            for key, value in response.headers.items():
                if key.lower() in ("content-type", "content-disposition", "cache-control"):
                    self.send_header(key, value)
            self.end_headers()
            self.wfile.write(response.read())


if __name__ == "__main__":
    if not (DIST / "index.html").is_file():
        raise SystemExit("Missing apps/web/dist/index.html")
    if not all((LANDING / file).is_file() for file in ("index.html", "login.html", "styles.css", "workspace-controls.js")):
        raise SystemExit("Missing landing page files")
    server = ThreadingHTTPServer(("127.0.0.1", 5173), Handler)
    print("Documi web: http://127.0.0.1:5173", flush=True)
    server.serve_forever()
