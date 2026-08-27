import Image from "@tiptap/extension-image";
import { TaskItem, TaskList } from "@tiptap/extension-list";
import Placeholder from "@tiptap/extension-placeholder";
import { TableKit } from "@tiptap/extension-table";
import { EditorContent, type Extensions, useEditor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import { useEffect } from "react";

import {
  createEmptyNativeDocument,
  type NativeDocument,
  nativeDocumentFormat,
} from "./native-format";
import "./styles.css";

export interface NativeDocumentEditorProps {
  initialDocument?: NativeDocument;
  readOnly?: boolean;
  extensions?: Extensions;
  onChange?: (document: NativeDocument) => void;
  onSave?: (document: NativeDocument) => void | Promise<void>;
}

export function NativeDocumentEditor({
  initialDocument = createEmptyNativeDocument(),
  readOnly = false,
  extensions = [],
  onChange,
  onSave,
}: NativeDocumentEditorProps) {
  const editor = useEditor({
    extensions: [
      StarterKit.configure({ heading: { levels: [1, 2, 3] } }),
      TaskList,
      TaskItem.configure({ nested: true }),
      TableKit.configure({ table: { resizable: true } }),
      Image.configure({ allowBase64: false }),
      Placeholder.configure({ placeholder: "Start writing…" }),
      ...extensions,
    ],
    content: initialDocument.content,
    editable: !readOnly,
    immediatelyRender: false,
    onUpdate: ({ editor: updatedEditor }) => {
      onChange?.({ format: nativeDocumentFormat, content: updatedEditor.getJSON() });
    },
  });

  useEffect(() => {
    editor?.setEditable(!readOnly);
  }, [editor, readOnly]);

  if (!editor) return <div className="documi-editor-loading">Loading editor…</div>;

  const save = () => onSave?.({ format: nativeDocumentFormat, content: editor.getJSON() });

  return (
    <section className="documi-editor" aria-label="Native Documi document editor">
      {!readOnly && (
        <div className="documi-editor-toolbar" role="toolbar" aria-label="Text formatting">
          <button
            type="button"
            aria-pressed={editor.isActive("bold")}
            onClick={() => editor.chain().focus().toggleBold().run()}
          >
            Bold
          </button>
          <button
            type="button"
            aria-pressed={editor.isActive("italic")}
            onClick={() => editor.chain().focus().toggleItalic().run()}
          >
            Italic
          </button>
          <button
            type="button"
            aria-pressed={editor.isActive("heading", { level: 2 })}
            onClick={() => editor.chain().focus().toggleHeading({ level: 2 }).run()}
          >
            Heading
          </button>
          <button
            type="button"
            aria-pressed={editor.isActive("bulletList")}
            onClick={() => editor.chain().focus().toggleBulletList().run()}
          >
            Bullets
          </button>
          <button
            type="button"
            aria-pressed={editor.isActive("orderedList")}
            onClick={() => editor.chain().focus().toggleOrderedList().run()}
          >
            Numbered
          </button>
          <button
            type="button"
            aria-pressed={editor.isActive("taskList")}
            onClick={() => editor.chain().focus().toggleTaskList().run()}
          >
            Tasks
          </button>
          <button
            type="button"
            aria-pressed={editor.isActive("codeBlock")}
            onClick={() => editor.chain().focus().toggleCodeBlock().run()}
          >
            Code
          </button>
          <button
            type="button"
            onClick={() =>
              editor.chain().focus().insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run()
            }
          >
            Table
          </button>
          <button
            type="button"
            disabled={!editor.can().undo()}
            onClick={() => editor.chain().focus().undo().run()}
          >
            Undo
          </button>
          <button
            type="button"
            disabled={!editor.can().redo()}
            onClick={() => editor.chain().focus().redo().run()}
          >
            Redo
          </button>
          {onSave && (
            <button className="documi-editor-save" type="button" onClick={save}>
              Save
            </button>
          )}
        </div>
      )}
      <EditorContent editor={editor} />
      {readOnly && (
        <p className="documi-editor-notice">You have read-only access to this document.</p>
      )}
    </section>
  );
}
