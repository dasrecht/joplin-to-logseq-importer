# Joplin to Logseq Importer

This is a simple script to import Joplin notes into Logseq.
I took inspiration from [this script](https://github.com/htinaunglu/Joplin-to-Logseq-Integrater).
It is a very simple script that takes a Joplin Markdown export and converts it to Logseq format.
Upon doing so we check if assets are existing and copy them over to output folder.

## Usage

0. You can run the application prior to having exported anything - it will create the necessary folders for you.
1. Export your Joplin notes as a `MD Markdown + Front Matter` file into a folder called `joplin-input`.
2. Run the script with `go run main.go`.
3. The script will create a `logseq-output` folder with the converted notes.
4. Copy the contents of the `logseq-output` folder into your Logseq notes folder.

## Features

- Frontmatter cleanup if you pass `--frontmatter-cleanup` flag.
  This will remove Latitude, Longitude, Altitude, Title, Author, and make the tags usable in logse from the frontmatter.

## Fair warning and warranty

*frantic laughter* - You have been warned - there is no [warranty or guarantee](LICENCE.md) this will work for you. It worked for me, but I make no promises.