"""Fixed, token-free LibreOffice commands for a Documi controlled working copy."""

import os
import subprocess
import sys
import urllib.parse

import unohelper
from com.sun.star.task import XJobExecutor


def _open_deep_link(action):
    working_copy_id = os.environ.get("DOCUMI_WORKING_COPY_ID", "")
    if not working_copy_id or any(ch not in "0123456789abcdef-" for ch in working_copy_id.lower()):
        return
    url = "documi://libreoffice/{}/{}".format(action, urllib.parse.quote(working_copy_id))
    if sys.platform == "darwin":
        subprocess.Popen(["open", url])
    elif os.name == "nt":
        os.startfile(url)  # pylint: disable=no-member
    else:
        subprocess.Popen(["xdg-open", url])


class DocumiCommands(unohelper.Base, XJobExecutor):
    def trigger(self, command):
        if command == "SaveToDocumi":
            _open_deep_link("save")
        elif command == "ShowInDocumi":
            _open_deep_link("show")


g_ImplementationHelper = unohelper.ImplementationHelper()
g_ImplementationHelper.addImplementation(
    DocumiCommands,
    "org.documi.libreoffice.DocumiCommands",
    ("com.sun.star.task.Job",),
)
