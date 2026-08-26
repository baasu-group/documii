from __future__ import annotations

import argparse
import json
import logging
import signal
import threading
from collections.abc import Sequence

from documi_ai.api.health import health
from documi_ai.config import Settings
from documi_ai.observability import configure_logging


def run(worker_name: str, arguments: Sequence[str] | None = None) -> None:
    parser = argparse.ArgumentParser(prog=worker_name)
    parser.add_argument(
        "--once",
        action="store_true",
        help="validate configuration, report health, and exit without polling",
    )
    args = parser.parse_args(arguments)
    settings = Settings.from_environment()
    configure_logging(settings.log_level)
    logger = logging.getLogger(worker_name)
    logger.info("worker configured", extra={"worker": worker_name})

    if args.once:
        payload = health(settings) | {"worker": worker_name}
        print(json.dumps(payload, sort_keys=True))
        return

    stop = threading.Event()
    for signal_name in (signal.SIGINT, signal.SIGTERM):
        signal.signal(signal_name, lambda *_: stop.set())
    logger.warning(
        "bootstrap worker is idle; Temporal activity polling is introduced with the first pipeline"
    )
    stop.wait()
    logger.info("worker stopped")
