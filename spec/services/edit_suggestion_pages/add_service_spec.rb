# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe EditSuggestionPages::AddService do
  it "編集提案に新しいページを追加すること" do
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record:)
    space_member_record = FactoryBot.create(:space_member_record, space_record:)
    edit_suggestion_record = FactoryBot.create(:edit_suggestion_record, :draft, space_record:, topic_record:, created_space_member_record: space_member_record)

    service = EditSuggestionPages::AddService.new
    result = service.call(
      edit_suggestion_record:,
      space_member_record:,
      page_record: nil,
      page_title: "追加されたページ",
      page_body: "# 追加されたページ\n\n追加されたページの内容です。"
    )

    expect(result.edit_suggestion_page_record).to be_persisted
    expect(result.edit_suggestion_page_record.edit_suggestion_record).to eq(edit_suggestion_record)
    expect(result.edit_suggestion_page_record.page_record).to be_nil
    expect(result.edit_suggestion_page_record.latest_revision_record.title).to eq("追加されたページ")
    expect(result.edit_suggestion_page_record.latest_revision_record.body).to eq("# 追加されたページ\n\n追加されたページの内容です。")
  end

  it "既存の編集提案ページを更新すること" do
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record:)
    space_member_record = FactoryBot.create(:space_member_record, space_record:)
    page_record = FactoryBot.create(:page_record, space_record:, topic_record:)
    edit_suggestion_record = FactoryBot.create(:edit_suggestion_record, :draft, space_record:, topic_record:, created_space_member_record: space_member_record)
    edit_suggestion_page_record = FactoryBot.build(:edit_suggestion_page_record, space_record:, edit_suggestion_record:, page_record:)
    edit_suggestion_page_record.save!(validate: false)
    # 初回のリビジョンを作成しておく
    initial_revision = FactoryBot.create(:edit_suggestion_page_revision_record, edit_suggestion_page_record:, space_record:)
    edit_suggestion_page_record.update!(latest_revision_record: initial_revision)

    service = EditSuggestionPages::AddService.new
    result = service.call(
      edit_suggestion_record:,
      space_member_record:,
      page_record:,
      page_title: "更新されたタイトル",
      page_body: "# 更新されたタイトル\n\n更新された内容です。"
    )

    expect(result.edit_suggestion_page_record).to eq(edit_suggestion_page_record)
    expect(result.edit_suggestion_page_record.latest_revision_record.title).to eq("更新されたタイトル")
    expect(result.edit_suggestion_page_record.latest_revision_record.body).to eq("# 更新されたタイトル\n\n更新された内容です。")
  end
end
