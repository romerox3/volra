import os
from langgraph.graph import StateGraph

API_KEY = os.environ["ANTHROPIC_API_KEY"]
DB_URL = os.environ.get("DATABASE_URL")

def build_graph():
    graph = StateGraph()
    return graph.compile()
