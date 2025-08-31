# typed: false
# frozen_string_literal: true

require_relative "shared_helpers"

RSpec.describe "Markdownエディター/Wikiリンクの補完候補", type: :system do
  include MarkdownEditorHelpers

  it "Wikiリンクの補完候補が表示されること" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record)
    space_member_record = create(:space_member_record, space_record:, user_record:)
    topic_record = create(:topic_record, space_record:)
    page_record = create(:page_record, space_record:, topic_record:)
    create(:topic_member_record, space_record:, topic_record:, space_member_record:)
    create(:page_record, :published, space_record:, topic_record:, title: "Other Page 1")
    create(:page_record, :published, space_record:, topic_record:, title: "Other Page 2")

    sign_in(user_record:)
    visit "/s/#{space_record.identifier}/pages/#{page_record.number}/edit"

    fill_in_editor(text: "[[Page")

    autocomplete_element = find(".cm-tooltip-autocomplete")
    visible_texts = autocomplete_element.find_css(".cm-completionLabel").map(&:visible_text)

    expect(visible_texts).to eq([
      "#{topic_record.name}/Other Page 2",
      "#{topic_record.name}/Other Page 1"
    ])
  end
end
