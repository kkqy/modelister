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

  it("按时间合并同一时间点的变动，并在时间轴项目内按供应商和 Key 分组", async () => {
    api.listModelChanges.mockResolvedValue({
      groups: [
        {
          id: 2,
          created_at: "2026-07-04T10:00:00Z",
          added_count: 4,
          removed_count: 3,
          providers: [
            {
              provider_id: 1,
              provider_name: "OpenAI",
              base_url: "https://api.openai.com",
              added_count: 3,
              removed_count: 3,
              keys: [
                {
                  id: 4,
                  key_id: 11,
                  key_name: "生产 Key",
                  added_count: 1,
                  removed_count: 1,
                  added_models: ["gpt-4o"],
                  removed_models: ["gpt-4-legacy"],
                  created_at: "2026-07-04T10:00:00Z",
                },
                {
                  id: 2,
                  key_id: 12,
                  key_name: "备用 Key",
                  added_count: 2,
                  removed_count: 2,
                  added_models: ["gpt-4.1", "gpt-4.1-mini"],
                  removed_models: ["gpt-3.5-turbo", "gpt-4"],
                  created_at: "2026-07-04T10:00:00Z",
                },
              ],
            },
            {
              provider_id: 2,
              provider_name: "Anthropic",
              base_url: "https://api.anthropic.com",
              added_count: 1,
              removed_count: 0,
              keys: [
                {
                  id: 3,
                  key_id: 21,
                  key_name: "默认 Key",
                  added_count: 1,
                  removed_count: 0,
                  added_models: ["claude-sonnet-4"],
                  removed_models: [],
                  created_at: "2026-07-04T10:00:00Z",
                },
              ],
            },
          ],
        },
        {
          id: 1,
          created_at: "2026-07-04T09:58:00Z",
          added_count: 1,
          removed_count: 0,
          providers: [
            {
              provider_id: 3,
              provider_name: "Mistral",
              base_url: "https://api.mistral.ai",
              added_count: 1,
              removed_count: 0,
              keys: [
                {
                  id: 1,
                  key_id: 31,
                  key_name: "欧洲 Key",
                  added_count: 1,
                  removed_count: 0,
                  added_models: ["mistral-large"],
                  removed_models: [],
                  created_at: "2026-07-04T09:58:00Z",
                },
              ],
            },
          ],
        },
      ],
      has_more: false,
      next_before_id: null,
    });

    const { container } = render(<ModelChanges toast={toast} onUnauthorized={vi.fn()} />);

    const firstTimelineSummary = await screen.findByText("2 个供应商 / 3 条 Key 变动");
    expect(container.querySelectorAll(".timeline-item")).toHaveLength(2);

    const group = firstTimelineSummary.closest(".timeline-content");
    const timelineSummary = group.querySelector(".timeline-head + .change-summary");
    expect(within(timelineSummary).getByText("+4 新增")).toBeInTheDocument();
    expect(within(timelineSummary).getByText("-3 移除")).toBeInTheDocument();
    expect(within(group).getByRole("heading", { name: "OpenAI" })).toBeInTheDocument();
    expect(within(group).getByRole("heading", { name: "Anthropic" })).toBeInTheDocument();

    const openAIGroup = within(group).getByRole("heading", { name: "OpenAI" }).closest(
      ".change-provider-group"
    );
    expect(within(openAIGroup).getByText("2 条 Key 变动")).toBeInTheDocument();
    expect(within(openAIGroup).getByText("+3 新增")).toBeInTheDocument();
    expect(within(openAIGroup).getByText("-3 移除")).toBeInTheDocument();
    expect(within(openAIGroup).getByText("生产 Key")).toBeInTheDocument();
    expect(within(openAIGroup).getByText("备用 Key")).toBeInTheDocument();
    expect(within(group).getByText("默认 Key")).toBeInTheDocument();
  });
});
