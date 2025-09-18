# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe EditSuggestion do
  it "draft状態のとき、draft?がtrueを返すこと" do
    space = Space.new(
      database_id: "space_id",
      identifier: "test-space",
      name: "スペース",
      plan: Plan::Free,
      joined_at: Time.current,
      can_create_topic: true
    )
    topic = Topic.new(
      database_id: "topic_id",
      number: 1,
      name: "トピック",
      description: "説明",
      visibility: TopicVisibility::Public,
      can_update: true,
      can_create_page: true,
      space:
    )
    user = User.new(
      database_id: "user_id",
      email: "user@example.com",
      atname: "@user",
      name: "ユーザー",
      description: "テストユーザー",
      locale: Locale::Ja,
      time_zone: "Asia/Tokyo"
    )
    edit_suggestion = EditSuggestion.new(
      database_id: "id",
      title: "タイトル",
      description: "説明",
      status: EditSuggestionStatus::Draft,
      applied_at: nil,
      created_at: Time.current,
      updated_at: Time.current,
      space:,
      topic:,
      created_user: user
    )

    expect(edit_suggestion.draft?).to be true
    expect(edit_suggestion.open?).to be false
    expect(edit_suggestion.applied?).to be false
    expect(edit_suggestion.closed?).to be false
  end

  it "open状態のとき、open?がtrueを返すこと" do
    space = Space.new(
      database_id: "space_id",
      identifier: "test-space",
      name: "スペース",
      plan: Plan::Free,
      joined_at: Time.current,
      can_create_topic: true
    )
    topic = Topic.new(
      database_id: "topic_id",
      number: 1,
      name: "トピック",
      description: "説明",
      visibility: TopicVisibility::Public,
      can_update: true,
      can_create_page: true,
      space:
    )
    user = User.new(
      database_id: "user_id",
      email: "user@example.com",
      atname: "@user",
      name: "ユーザー",
      description: "テストユーザー",
      locale: Locale::Ja,
      time_zone: "Asia/Tokyo"
    )
    edit_suggestion = EditSuggestion.new(
      database_id: "id",
      title: "タイトル",
      description: "説明",
      status: EditSuggestionStatus::Open,
      applied_at: nil,
      created_at: Time.current,
      updated_at: Time.current,
      space:,
      topic:,
      created_user: user
    )

    expect(edit_suggestion.draft?).to be false
    expect(edit_suggestion.open?).to be true
    expect(edit_suggestion.applied?).to be false
    expect(edit_suggestion.closed?).to be false
  end

  it "applied状態のとき、applied?がtrueを返すこと" do
    space = Space.new(
      database_id: "space_id",
      identifier: "test-space",
      name: "スペース",
      plan: Plan::Free,
      joined_at: Time.current,
      can_create_topic: true
    )
    topic = Topic.new(
      database_id: "topic_id",
      number: 1,
      name: "トピック",
      description: "説明",
      visibility: TopicVisibility::Public,
      can_update: true,
      can_create_page: true,
      space:
    )
    user = User.new(
      database_id: "user_id",
      email: "user@example.com",
      atname: "@user",
      name: "ユーザー",
      description: "テストユーザー",
      locale: Locale::Ja,
      time_zone: "Asia/Tokyo"
    )
    edit_suggestion = EditSuggestion.new(
      database_id: "id",
      title: "タイトル",
      description: "説明",
      status: EditSuggestionStatus::Applied,
      applied_at: Time.current,
      created_at: Time.current,
      updated_at: Time.current,
      space:,
      topic:,
      created_user: user
    )

    expect(edit_suggestion.draft?).to be false
    expect(edit_suggestion.open?).to be false
    expect(edit_suggestion.applied?).to be true
    expect(edit_suggestion.closed?).to be false
  end

  it "closed状態のとき、closed?がtrueを返すこと" do
    space = Space.new(
      database_id: "space_id",
      identifier: "test-space",
      name: "スペース",
      plan: Plan::Free,
      joined_at: Time.current,
      can_create_topic: true
    )
    topic = Topic.new(
      database_id: "topic_id",
      number: 1,
      name: "トピック",
      description: "説明",
      visibility: TopicVisibility::Public,
      can_update: true,
      can_create_page: true,
      space:
    )
    user = User.new(
      database_id: "user_id",
      email: "user@example.com",
      atname: "@user",
      name: "ユーザー",
      description: "テストユーザー",
      locale: Locale::Ja,
      time_zone: "Asia/Tokyo"
    )
    edit_suggestion = EditSuggestion.new(
      database_id: "id",
      title: "タイトル",
      description: "説明",
      status: EditSuggestionStatus::Closed,
      applied_at: nil,
      created_at: Time.current,
      updated_at: Time.current,
      space:,
      topic:,
      created_user: user
    )

    expect(edit_suggestion.draft?).to be false
    expect(edit_suggestion.open?).to be false
    expect(edit_suggestion.applied?).to be false
    expect(edit_suggestion.closed?).to be true
  end

  it "draftまたはopen状態のとき、editable?がtrueを返すこと" do
    space = Space.new(
      database_id: "space_id",
      identifier: "test-space",
      name: "スペース",
      plan: Plan::Free,
      joined_at: Time.current,
      can_create_topic: true
    )
    topic = Topic.new(
      database_id: "topic_id",
      number: 1,
      name: "トピック",
      description: "説明",
      visibility: TopicVisibility::Public,
      can_update: true,
      can_create_page: true,
      space:
    )
    user = User.new(
      database_id: "user_id",
      email: "user@example.com",
      atname: "@user",
      name: "ユーザー",
      description: "テストユーザー",
      locale: Locale::Ja,
      time_zone: "Asia/Tokyo"
    )

    draft = EditSuggestion.new(
      database_id: "id",
      title: "タイトル",
      description: "説明",
      status: EditSuggestionStatus::Draft,
      applied_at: nil,
      created_at: Time.current,
      updated_at: Time.current,
      space:,
      topic:,
      created_user: user
    )

    open = EditSuggestion.new(
      database_id: "id",
      title: "タイトル",
      description: "説明",
      status: EditSuggestionStatus::Open,
      applied_at: nil,
      created_at: Time.current,
      updated_at: Time.current,
      space:,
      topic:,
      created_user: user
    )

    applied = EditSuggestion.new(
      database_id: "id",
      title: "タイトル",
      description: "説明",
      status: EditSuggestionStatus::Applied,
      applied_at: Time.current,
      created_at: Time.current,
      updated_at: Time.current,
      space:,
      topic:,
      created_user: user
    )

    expect(draft.editable?).to be true
    expect(open.editable?).to be true
    expect(applied.editable?).to be false
  end
end
