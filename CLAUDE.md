# CLAUDE.md

To workaround lack of support for AGENTS.md standard, claude must import and read ./AGENTS.md in order to have one source-of-truth guidance for all agents including Claude for guidance when working with code in this repository.  This file should only contain claude-specific directions that arent supported by other Agents (skills/plugins/etc)

@AGENTS.md
@CONTRIBUTING.md

## Handling AGENTS.md in sub-folders

- Dont read AGENTS.md in a subfolder if you arent working in that folder.  Don't read all AGENTS.md at session start.
- If you will need to modify or research files in a subfolder, only then must you read any AGENTS.md files in the subfolder for additional context and direction. 
- If there is conflicting guidance - prefer the one defined in the AGENTS.md in the lowest subfolder

## Claude Specific Directions

 - After `/compact` skill is used (or autocompaction is initiated). Prompt the user with some proposed edits to AGENTS.md that would help reduce token or context window usage for future sessions.