import json

from documi_ai import __version__
from documi_ai.config import Settings


def health(settings: Settings) -> dict[str, object]:
    return {
        "status": "ok",
        "service": "documi-ai",
        "version": __version__,
        "config": settings.public_dict(),
    }


def main() -> None:
    print(json.dumps(health(Settings.from_environment()), sort_keys=True))
