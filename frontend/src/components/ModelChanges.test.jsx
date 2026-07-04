import React from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, within } from "@testing-library/react";
import ModelChanges from "./ModelChanges.jsx";
import { api } from "../api.js";

vi.mock("../api.js", async () => {
  const actual = await vi.importActual("../api.js");
  return {
    ...actual,
    api: {
      listModelChanges: vi.fn(),
    },
  };
});

describe("ModelChanges", () => {
  const toast = {
    error: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("按照供应商分组展示同一供应商下不同 Key 的变动记录", async () => {
    api.listModelChanges.mockResolvedValue({
      events: [
        {
          id: 3,
          provider_id: 1,
          provider_key_id: 11,
          provider_name: "OpenAI",
          key_name: "生产 Key",
          base_url: "https://api.openai.com",
          added_count: 1,
          removed_count: 1,
          added_models: ["gpt-4o"],
          removed_models: ["gpt-4-legacy"],
          created_at: "2026-07-04T10:00:00Z",
        },
        {
          id: 2,
          provider_id: 1,
          provider_key_id: 12,
          provider_name: "OpenAI",
          key_name: "备用 Key",
          base_url: "https://api.openai.com",
          added_count: 2,
          removed_count: 2,
          added_models: ["gpt-4.1", "gpt-4.1-mini"],
          removed_models: ["gpt-3.5-turbo", "gpt-4"],
          created_at: "2026-07-04T09:59:00Z",
        },
        {
          id: 1,
          provider_id: 2,
          provider_key_id: 21,
          provider_name: "Anthropic",
          key_name: "默认 Key",
          base_url: "https://api.anthropic.com",
          added_count: 1,
          removed_count: 0,
          added_models: ["claude-sonnet-4"],
          removed_models: [],
          created_at: "2026-07-04T09:58:00Z",
        },
      ],
      has_more: false,
      next_before_id: null,
    });

    render(<ModelChanges toast={toast} onUnauthorized={vi.fn()} />);

    const openAIGroup = await screen.findByRole("heading", { name: "OpenAI" });
    expect(screen.getAllByRole("heading", { name: "OpenAI" })).toHaveLength(1);
    expect(screen.getByRole("heading", { name: "Anthropic" })).toBeInTheDocument();

    const group = openAIGroup.closest(".timeline-content");
    expect(within(group).getByText("2 条 Key 变动")).toBeInTheDocument();
    expect(within(group).getByText("+3 新增")).toBeInTheDocument();
    expect(within(group).getByText("-3 移除")).toBeInTheDocument();
    expect(within(group).getByText("生产 Key")).toBeInTheDocument();
    expect(within(group).getByText("备用 Key")).toBeInTheDocument();
  });
});
