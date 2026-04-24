# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Module: `github.com/javorszky/hoplink-go`  
Language: Go 1.26.2

This project is in its earliest stage — only `go.mod` exists. Architecture and conventions will grow here as the codebase develops.

## Domain guidelines

Load only the file(s) relevant to the task at hand.

- **Overall system design, API contract, decoupling rules**: see [`.ai/architecture/overview.md`](.ai/architecture/overview.md)
- **Backend (Go, Echo, OTel, coding style, context, shutdown)**: see [`.ai/backend/guidelines.md`](.ai/backend/guidelines.md)
- **Frontend (Vue 3, Reka UI, Tailwind CSS v4, Vite)**: see [`.ai/frontend/guidelines.md`](.ai/frontend/guidelines.md)
