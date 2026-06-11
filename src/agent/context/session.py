import json as J
from ..config import CONFIG
from human_id import generate_id


class Session:
    def __init__(self, session_id: str | None = None):
        self.session_id = session_id if session_id else self._find_last_session_id()
        self._initialize()

    @property
    def session_file_path(self):
        return CONFIG.WORKSPACE_DIR / "sessions" / f"{self.session_id}.jsonl"

    @property
    def last_session_id(self):
        return CONFIG.WORKSPACE_DIR / "sessions" / "last_session_id.txt"

    def _initialize(self):
        CONFIG.WORKSPACE_DIR.mkdir(parents=True, exist_ok=True)
        (CONFIG.WORKSPACE_DIR / "sessions").mkdir(parents=True, exist_ok=True)
        if not self.session_file_path.exists():
            self.session_file_path.touch()

        self.last_session_id.write_text(self.session_id)

    def _find_last_session_id(self):
        if self.last_session_id.exists():
            return self.last_session_id.read_text().strip()
        return self._generate_new_session_id()

    def _generate_new_session_id(self):
        return generate_id()

    def _append_message_to_session(self, message):
        with open(self.session_file_path, "a") as file:
            file.write(J.dumps(message) + "\n")

    def _overwrite_messages_in_session(self, messages):
        with open(self.session_file_path, "w") as file:
            for message in messages:
                file.write(J.dumps(message) + "\n")

    def _load_messages_from_session(self):
        with open(self.session_file_path, "r") as file:
            return [J.loads(line) for line in file if line.strip()]
