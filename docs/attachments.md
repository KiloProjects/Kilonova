# Kilonova Problem Attachments

Attachments are files associated with a problem. They are the backbone of a problem. 

Some uses for attachments are:
- Special attachments influence the display and the execution behaviour of the problem;
- Images to attach onto the problem markdown;
- Provide sample code for interactive problems.

Attachments of the following formats are considered special:

- `statement-{language}.{format}`;
- `grader.{extension}`.
- `editorial-{language}.format`

Where:
- `language`: Display language of the statement/editorial. Can be any of `en` (English) and `ro` (Romanian);
- `format`: Any of `md` (markdown), `pdf` (PDF file). More formats might be supported in the future;
- `extension`: A valid extension for a source code file. For example, `cpp` for C++, `c` for C, `py` for Python, etc.

# Attachment properties

All attachments have the following properties: 
- Visible: Attachment that can be seen in the sidebar of a problem.
- Private: Attachment that cannot be downloaded by a user.

Please note that the existence of both visible and private attachments can be seen via an API request. Only non-private attachments can, though, be downloaded by a user.

Private attachments should not be visible, but they can be made visible if the problem author deems fit (for example, to mark the existence of a grader).
