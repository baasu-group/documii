from collections.abc import Sequence

from .runtime import run


def main(arguments: Sequence[str] | None = None) -> None:
    run("ai-worker", arguments)


if __name__ == "__main__":
    main()
