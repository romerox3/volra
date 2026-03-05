"""Tests for the embedded pricing table."""

from volra_observe.pricing import COST_PER_1K_TOKENS, estimate_cost


def test_known_model_cost():
    """GPT-4o cost calculation matches expected values."""
    cost = estimate_cost("gpt-4o", input_tokens=1000, output_tokens=1000)
    expected = (1000 * 0.005 + 1000 * 0.015) / 1000  # $0.02
    assert abs(cost - expected) < 1e-9


def test_unknown_model_returns_zero():
    """Unknown models return 0 cost (graceful degradation)."""
    assert estimate_cost("unknown-model-xyz", 1000, 1000) == 0.0


def test_zero_tokens():
    """Zero tokens = zero cost."""
    assert estimate_cost("gpt-4o", 0, 0) == 0.0


def test_anthropic_model():
    """Anthropic model cost calculation."""
    cost = estimate_cost("claude-sonnet-4-20250514", input_tokens=2000, output_tokens=500)
    expected = (2000 * 0.003 + 500 * 0.015) / 1000
    assert abs(cost - expected) < 1e-9


def test_all_models_have_input_output():
    """Every model in the pricing table has both input and output keys."""
    for model, prices in COST_PER_1K_TOKENS.items():
        assert "input" in prices, f"{model} missing input price"
        assert "output" in prices, f"{model} missing output price"
        assert prices["input"] >= 0, f"{model} has negative input price"
        assert prices["output"] >= 0, f"{model} has negative output price"
