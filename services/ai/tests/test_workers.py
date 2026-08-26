import json

import pytest

from documi_ai.workers.document import main


def test_document_worker_once(capsys: pytest.CaptureFixture[str]) -> None:
    main(["--once"])
    payload = json.loads(capsys.readouterr().out)
    assert payload["worker"] == "document-worker"
    assert payload["status"] == "ok"
