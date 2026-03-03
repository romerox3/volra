import os

DB_HOST = os.getenv("DB_HOST", "localhost")

def main():
    print("Running agent")

if __name__ == "__main__":
    main()
