# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe EditSuggestions::CreateService do
  it "新規ページの編集提案を作成すること" do
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record:)
    space_member_record = FactoryBot.create(:space_member_record, space_record:)
    # 編集提案はページを起点に作成される
    page_record = FactoryBot.create(:page_record, space_record:, topic_record:)

    service = EditSuggestions::CreateService.new
    result = service.call(
      space_member_record:,
      page_record:,
      title: "新機能の提案",
      description: "新しい機能を追加します",
      page_title: "新しいページ",
      page_body: "# 新しいページ\n\nこれは新しいページの内容です。"
    )

    expect(result.edit_suggestion_record).to be_persisted
    expect(result.edit_suggestion_record.title).to eq("新機能の提案")
    expect(result.edit_suggestion_record.description).to eq("新しい機能を追加します")
    expect(result.edit_suggestion_record.status_draft?).to be(true)
    expect(result.edit_suggestion_record.created_space_member_record).to eq(space_member_record)
    expect(result.edit_suggestion_record.topic_record).to eq(topic_record)

    expect(result.edit_suggestion_page_record).to be_persisted
    expect(result.edit_suggestion_page_record.page_record).to eq(page_record)
    expect(result.edit_suggestion_page_record.latest_revision_record.title).to eq("新しいページ")
    expect(result.edit_suggestion_page_record.latest_revision_record.body).to eq("# 新しいページ\n\nこれは新しいページの内容です。")
    expect(result.edit_suggestion_page_record.latest_revision_record.body_html).to include("新しいページ</h1>")
  end

  it "既存ページの編集提案を作成すること" do
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record:)
    space_member_record = FactoryBot.create(:space_member_record, space_record:)
    page_record = FactoryBot.create(:page_record, space_record:, topic_record:)
    page_revision_record = FactoryBot.create(:page_revision_record, page_record:)

    service = EditSuggestions::CreateService.new
    result = service.call(
      space_member_record:,
      page_record:,
      title: "ページの更新提案",
      description: "既存ページを更新します",
      page_title: "更新後のタイトル",
      page_body: "# 更新後のタイトル\n\n更新された内容です。"
    )

    expect(result.edit_suggestion_record).to be_persisted
    expect(result.edit_suggestion_page_record).to be_persisted
    expect(result.edit_suggestion_page_record.page_record).to eq(page_record)
    expect(result.edit_suggestion_page_record.page_revision_record).to eq(page_revision_record)
  end
end
