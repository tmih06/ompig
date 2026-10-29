# Applied use cases

These walkthroughs show Ompig doing everyday work that is not programming: summarising documents, cleaning a spreadsheet, drafting text, producing a weekly report and tidying files. Each one gives the exact prompts to type, what Ompig shows on screen, how to check the result and how to undo it.

You do not need to know how to program. You need a terminal, a folder of files and a model that Ompig can use. If you have not set those up, follow [Your first ten minutes](/docs/latest/first-run) first.

## The walkthroughs

| Task | Page | Changes your files? |
|---|---|---|
| Read a folder of PDFs and write a summary | [Summarise a folder of PDFs](/docs/latest/usecase-summarise-pdfs) | Adds one file |
| Clean a messy CSV export | [Clean a spreadsheet](/docs/latest/usecase-clean-spreadsheet) | Adds a cleaned copy and a change list |
| Write a draft from notes, then edit one part | [Draft and edit a document](/docs/latest/usecase-draft-document) | Adds a draft, then edits it |
| Produce the same report every week, unattended | [Automate a weekly report](/docs/latest/usecase-weekly-report) | Adds one script and one schedule line |
| Sort a messy folder into tidy ones | [Organise files](/docs/latest/usecase-organise-files) | Moves files, with an undo log |

## How each walkthrough is laid out

1. **Before you start** lists what you need.
2. **Steps** give each prompt as a text block. Type it into Ompig exactly as written, then press `Enter`.
3. **What Ompig shows** is a screen taken from a real Ompig session. Each walkthrough was run against Ompig with a scripted stand-in model, so the commands, file contents and layout are what Ompig printed. The model's own wording will differ on your computer, and so may the exact commands it picks. The shape stays the same: Ompig prints each tool it runs, then the model's reply.
4. **Check** tells you how to confirm the result yourself, without trusting the model's description of it.
5. **Undo** tells you how to get back to where you started.

## Before you start: how Ompig works on your files

Read this once. It is short, and it is the reason the walkthroughs ask for plans and copies.

### Ompig acts without asking

Ompig runs commands and writes files with your own permissions. It does not stop to ask before it does. When the model decides to run a command or write a file, it happens at once, and the screen shows it afterwards. This follows Pi: the `write` tool creates or overwrites a file directly (`packages/coding-agent/src/core/tools/write.ts:78-82` in Pi), and Ompig is not a security sandbox ([Security](/docs/latest/security)).

The tools also accept absolute paths, so Ompig is not limited to the folder you start it in. Start Ompig inside the folder that holds the work, and keep other things out of that folder.

### Your files go to the model provider

Everything Ompig reads is sent to the provider of the model you chose. That includes the text of a spreadsheet it opens and the output of every command it runs. Use a model and provider you are allowed to send the material to.

### Ompig saves the conversation

Ompig saves each conversation as a file under `~/.ompig/agent/sessions/`. The file holds what Ompig read and wrote. Start Ompig with `--no-session` to keep the conversation in memory only ([Sessions](/docs/latest/sessions)).

### Five habits that make undo possible

1. **Work on a copy.** Copy the folder, then start Ompig in the copy.

   ```bash
   cp -R ~/Documents/reports ~/Documents/reports-ompig
   cd ~/Documents/reports-ompig
   ompig
   ```

   On Windows PowerShell, use `Copy-Item -Recurse` in place of `cp -R`.

2. **Ask for a plan first.** Start with "Do not change anything yet" and read the plan. Then say "Do it".
3. **Ask for new files, not edits.** "Write the cleaned copy to `sales-clean.csv` and never change `sales.csv`" leaves the original safe.
4. **Ask for a log.** "Record every change in `changes.md`" gives you a list to check and to reverse.
5. **Never delete.** Say "Use `mv`, never delete" or "Do not delete anything". Nothing in Ompig brings back a deleted file.

### How to undo

| What Ompig did | How you undo it |
|---|---|
| Created a new file | Delete that file. `ls -lt` lists the newest files first. |
| Moved or renamed files | Run the undo commands from the log you asked for. See [Organise files](/docs/latest/usecase-organise-files). |
| Edited a file you did not copy | Ask Ompig to put the old text back. It can read the change it made from the conversation. A copy kept beforehand (`update.v1.md`) is more reliable. |
| Deleted a file | Restore it from your own backup or from your trash. Ompig cannot. |
| Said something wrong | Press `Esc` to stop the model, then type a correction. `/tree` and `/fork` go back in the **conversation** only. They do not undo changes to files. |

If you use Git, a snapshot before you start gives a complete undo: `git init`, `git add -A`, `git commit -m "before Ompig"`. Afterwards `git status` lists what changed, and `git restore .` brings back every edited or deleted tracked file.

### How to stop

Press `Esc` while Ompig works. It cancels the model's turn and any command it is running. Changes already made stay made. Press `Ctrl+C` to clear what you typed, and press it again within half a second to leave Ompig. `Ctrl+D` leaves when the editor is empty ([Using Ompig](/docs/latest/usage)).

## If Ompig feels like too much

You can ignore nearly all of it. Four things are enough for every walkthrough here:

1. **Type what you want in plain words and press `Enter`.** Ompig shows what it does as it goes.
2. **Press `Esc` to stop it.** Changes already made stay made.
3. **Type `/` to see the commands.** `/session` shows what the conversation has used. `/quit` leaves.
4. **Ask Ompig about Ompig.** The startup screen says "Ompig can explain its own features and look up its docs. Ask it how to use or extend Ompig." Type a question such as `How do I pick up yesterday's conversation again?`. Name the task, not the screen: "go back to an earlier point" got an answer about scrolling. Ompig carries a copy of its manual and reads it to answer (`ompig docs list` lists the pages).

Everything else, the models, settings, skills and extensions, is optional. Start with the one walkthrough that matches a task you already do, and leave the rest.

## What to expect from the model

Ompig is a harness around a model. It does not make the model correct. Models misread a column, miscount rows and describe work they did not do. Open the file Ompig wrote and compare it with your source before you use it, and ask the model to show its working ("List every change you made") when the result matters.
