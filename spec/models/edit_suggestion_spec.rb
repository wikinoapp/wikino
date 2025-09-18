# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe EditSuggestion do
  it "draft状態のとき、draft?がtrueを返すこと" do
    edit_suggestion = EditSuggestion.new(
      id: "id",
      space_id: "space_id",
      topic_id: "topic_id",
      created_user_id: "user_id",
      title: "タイトル",
      description: "説明",
      status: "draft",
      applied_at: nil,
      created_at: Time.current,
      updated_at: Time.current
    )

    expect(edit_suggestion.draft?).to be true
    expect(edit_suggestion.open?).to be false
    expect(edit_suggestion.applied?).to be false
    expect(edit_suggestion.closed?).to be false
  end

  it "open状態のとき、open?がtrueを返すこと" do
    edit_suggestion = EditSuggestion.new(
      id: "id",
      space_id: "space_id",
      topic_id: "topic_id",
      created_user_id: "user_id",
      title: "タイトル",
      description: "説明",
      status: "open",
      applied_at: nil,
      created_at: Time.current,
      updated_at: Time.current
    )

    expect(edit_suggestion.draft?).to be false
    expect(edit_suggestion.open?).to be true
    expect(edit_suggestion.applied?).to be false
    expect(edit_suggestion.closed?).to be false
  end

  it "applied状態のとき、applied?がtrueを返すこと" do
    edit_suggestion = EditSuggestion.new(
      id: "id",
      space_id: "space_id",
      topic_id: "topic_id",
      created_user_id: "user_id",
      title: "タイトル",
      description: "説明",
      status: "applied",
      applied_at: Time.current,
      created_at: Time.current,
      updated_at: Time.current
    )

    expect(edit_suggestion.draft?).to be false
    expect(edit_suggestion.open?).to be false
    expect(edit_suggestion.applied?).to be true
    expect(edit_suggestion.closed?).to be false
  end

  it "closed状態のとき、closed?がtrueを返すこと" do
    edit_suggestion = EditSuggestion.new(
      id: "id",
      space_id: "space_id",
      topic_id: "topic_id",
      created_user_id: "user_id",
      title: "タイトル",
      description: "説明",
      status: "closed",
      applied_at: nil,
      created_at: Time.current,
      updated_at: Time.current
    )

    expect(edit_suggestion.draft?).to be false
    expect(edit_suggestion.open?).to be false
    expect(edit_suggestion.applied?).to be false
    expect(edit_suggestion.closed?).to be true
  end

  it "draftまたはopen状態のとき、editable?がtrueを返すこと" do
    draft = EditSuggestion.new(
      id: "id",
      space_id: "space_id",
      topic_id: "topic_id",
      created_user_id: "user_id",
      title: "タイトル",
      description: "説明",
      status: "draft",
      applied_at: nil,
      created_at: Time.current,
      updated_at: Time.current
    )

    open = EditSuggestion.new(
      id: "id",
      space_id: "space_id",
      topic_id: "topic_id",
      created_user_id: "user_id",
      title: "タイトル",
      description: "説明",
      status: "open",
      applied_at: nil,
      created_at: Time.current,
      updated_at: Time.current
    )

    applied = EditSuggestion.new(
      id: "id",
      space_id: "space_id",
      topic_id: "topic_id",
      created_user_id: "user_id",
      title: "タイトル",
      description: "説明",
      status: "applied",
      applied_at: Time.current,
      created_at: Time.current,
      updated_at: Time.current
    )

    expect(draft.editable?).to be true
    expect(open.editable?).to be true
    expect(applied.editable?).to be false
  end
end
