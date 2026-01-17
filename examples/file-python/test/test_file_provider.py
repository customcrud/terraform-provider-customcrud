import json
from pathlib import Path

import pytest
from click.testing import CliRunner

from src.file_provider import main

@pytest.fixture
def runner():
    return CliRunner()

def test_create(runner):
    with runner.isolated_filesystem():
        result = runner.invoke(main, ['--action=create'], input=json.dumps({"input": {"content": "test content"}}))
        assert result.exit_code == 0
        output = json.loads(result.output)
        assert output["content"] == "test content"
        assert Path(output["id"]).exists()
        assert Path(output["id"]).read_text() == "test content"

def test_read_success(runner):
    with runner.isolated_filesystem():
        Path("testfile").write_text("existing content")
        file_path = str(Path("testfile").resolve())
        
        result = runner.invoke(main, ['--action=read'], input=json.dumps({"id": file_path}))
        assert result.exit_code == 0
        output = json.loads(result.output)
        assert output["content"] == "existing content"
        assert output["id"] == file_path

def test_read_missing_file(runner):
    with runner.isolated_filesystem():
        result = runner.invoke(main, ['--action=read'], input=json.dumps({"id": "non_existent_file"}))
        assert result.exit_code == 22

def test_update(runner):
    with runner.isolated_filesystem():
        Path("testfile").write_text("old content")
        file_path = str(Path("testfile").resolve())

        result = runner.invoke(main, ['--action=update'], input=json.dumps({"id": file_path, "input": {"content": "new content"}}))
        assert result.exit_code == 0
        output = json.loads(result.output)
        assert output["content"] == "new content"
        
        assert Path(file_path).read_text() == "new content"

def test_delete(runner):
    with runner.isolated_filesystem():
        result = runner.invoke(main, ['--action=delete'], input=json.dumps({"id": "any"}))
        assert result.exit_code == 0
