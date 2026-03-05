from flask import Flask
app = Flask(__name__)

@app.route("/health")
def health():
    return {"ok": True}

if __name__ == "__main__":
    app.run(port=5000)
