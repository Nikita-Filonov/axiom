# 📘 Usage

This directory contains a **realistic, end-to-end example** of how Axiom can be used in practice to build a small,
composable test framework on top of Go’s native `testing` package.

The code in this folder demonstrates:

- how to define a **base (platform-level) runner**
- how to extend it with **domain-specific runners** using composition
- how to organize **fixtures, context, and metadata**
- how tests interact only with their **domain runner**, not the framework internals

This is **not a tutorial** and **not a full test suite**. Instead, it serves as a **reference architecture** showing how
teams can structure their own test platforms using Axiom.

If you are looking for a practical, idiomatic example beyond isolated API snippets, start here.
