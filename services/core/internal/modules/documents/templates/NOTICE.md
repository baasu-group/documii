# Office template asset provenance

The encoded files in `assets/` are unmodified binary fixtures from immutable upstream commits. Runtime code decodes them and verifies the listed SHA-256 before use.

| Asset | Source | License | SHA-256 |
| --- | --- | --- | --- |
| `blank.docx.b64` | [`python-openxml/python-docx` `doc-word-default-blank.docx` at `e454546`](https://github.com/python-openxml/python-docx/blob/e45454602b53e8e572b179ccf1c91093ec9f4ed7/features/steps/test_files/doc-word-default-blank.docx) | [MIT](https://github.com/python-openxml/python-docx/blob/e45454602b53e8e572b179ccf1c91093ec9f4ed7/LICENSE) | `560b0e299a592f6b7d1ec9abab98648f1d92bf8fa684ce544a0424fdde798903` |
| `blank.xlsx.b64` | [`dtjohnson/xlsx-populate` `blank/blank.xlsx` at `7480a02`](https://github.com/dtjohnson/xlsx-populate/blob/7480a02575c9140c0e7995623ea192c88c1886d3/blank/blank.xlsx) | [MIT](https://github.com/dtjohnson/xlsx-populate/blob/7480a02575c9140c0e7995623ea192c88c1886d3/LICENSE) | `32ce7d6f125c2cdde13a61c94c6fd0f05e96dea7ab73c0f7ee6b67b6dd37cf04` |
| `blank.pptx.b64` | [`office-kit/pptx` `test/fixtures/minimal/blank.pptx` at `51b1f90`](https://github.com/office-kit/pptx/blob/51b1f90b91ef9ea30b49bc38e5500c8acc344412/test/fixtures/minimal/blank.pptx) | [MIT](https://github.com/office-kit/pptx/blob/51b1f90b91ef9ea30b49bc38e5500c8acc344412/LICENSE) | `dd4746eb40918140f90310247c04e3a3aa7232689de2e2f37670a10e562a4454` |

The original project license texts remain authoritative. This notice records provenance; it does not relicense the upstream assets.
