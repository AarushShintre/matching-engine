// Package strategy implements a simulation market-making client for the
// matching-engine demo (Spec 6).
//
// This package exists only to demonstrate an event-driven feedback loop:
// market data → decision → Spec 2 ingestion → book/trade → market data again.
//
// It is not a profitable trading strategy, not production-ready automated
// trading, and not investment advice. Documentation, demos, comments, and
// resume material MUST describe it as a system-design simulation
// (Constitution Principle VI).
package strategy
