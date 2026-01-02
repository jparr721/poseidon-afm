#!/usr/bin/env -S uv run --script
#
# /// script
# requires-python = ">=3.12"
# dependencies = ["fastapi", "uvicorn"]
# ///
"""
Test backend for Poseidon local UI polling.

Usage:
    chmod +x test_backend.py
    ./test_backend.py

Or:
    uv run test_backend.py
"""

import logging
import uuid
from collections import defaultdict
from datetime import datetime
from typing import Any

import uvicorn
from fastapi import FastAPI, Header, Request

logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(message)s")
logger = logging.getLogger(__name__)

app = FastAPI(title="Poseidon Test Backend")

# In-memory storage
agents: dict[str, dict[str, Any]] = {}
task_queues: dict[str, list[dict[str, Any]]] = defaultdict(list)
task_counter = 0


@app.post("/checkin")
async def checkin(request: Request):
    """Accept agent check-in and return an agent ID."""
    body = await request.json()
    agent_id = str(uuid.uuid4())[:8]

    agents[agent_id] = {
        "id": agent_id,
        "checkin_time": datetime.now().isoformat(),
        "info": body,
    }

    logger.info(f"Agent checked in: {agent_id}")
    logger.info(f"  OS: {body.get('os', 'unknown')}")
    logger.info(f"  User: {body.get('user', 'unknown')}@{body.get('host', 'unknown')}")
    logger.info(f"  PID: {body.get('pid', 'unknown')}")

    return {"id": agent_id}


@app.post("/poll")
async def poll(request: Request, x_agent_id: str | None = Header(None)):
    """
    Accept poll request from agent, return queued tasks.

    Agent sends: MythicMessage with action="get_tasking" and any responses
    Backend returns: MythicMessageResponse with tasks array
    """
    body = await request.json()
    agent_id = x_agent_id or "unknown"

    # Log any responses the agent sent
    responses = body.get("responses", [])
    if responses:
        logger.info(f"Agent {agent_id} sent {len(responses)} response(s):")
        for resp in responses:
            task_id = resp.get("task_id", "?")
            status = resp.get("status", "?")
            logger.info(f"  Task {task_id}: status={status}")
            if "file_browser" in resp:
                fb = resp["file_browser"]
                logger.info(f"    file_browser: {len(fb.get('files', []))} files in {fb.get('parent_path', '?')}")
            if "user_output" in resp:
                output = resp["user_output"]
                preview = output[:100] + "..." if len(output) > 100 else output
                logger.info(f"    output: {preview}")

    # Pop any queued tasks for this agent
    tasks = task_queues[agent_id]
    task_queues[agent_id] = []

    if tasks:
        logger.info(f"Sending {len(tasks)} task(s) to agent {agent_id}")

    return {
        "action": "get_tasking",
        "tasks": tasks,
    }


@app.post("/queue_task")
async def queue_task(request: Request):
    """
    Helper endpoint to queue a task for an agent.

    Example:
        curl -X POST http://localhost:11111/queue_task \
          -H "Content-Type: application/json" \
          -d '{"agent_id":"abc123","command":"ls","parameters":"{\"path\":\".\",\"depth\":1}"}'
    """
    global task_counter
    body = await request.json()

    agent_id = body.get("agent_id")
    if not agent_id:
        return {"error": "agent_id required"}

    task_counter += 1
    task = {
        "id": f"task-{task_counter}",
        "command": body.get("command", ""),
        "parameters": body.get("parameters", "{}"),
        "timestamp": int(datetime.now().timestamp()),
    }

    task_queues[agent_id].append(task)
    logger.info(f"Queued task {task['id']} ({task['command']}) for agent {agent_id}")

    return {"status": "queued", "task": task}


@app.get("/agents")
async def list_agents():
    """List all registered agents."""
    return {"agents": list(agents.values())}


@app.get("/")
async def root():
    """Health check and usage info."""
    return {
        "status": "ok",
        "endpoints": {
            "POST /checkin": "Agent check-in",
            "POST /poll": "Agent poll for tasks",
            "POST /queue_task": "Queue a task for an agent",
            "GET /agents": "List all agents",
        },
    }


if __name__ == "__main__":
    logger.info("Starting Poseidon test backend on http://localhost:11111")
    uvicorn.run(app, host="0.0.0.0", port=11111)
