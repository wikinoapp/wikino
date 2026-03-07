# typed: false
# frozen_string_literal: true

RSpec.describe DraftPage, type: :model do
  describe "#display_title" do
    it "下書きのタイトルがある場合、下書きのタイトルを返すこと" do
      draft_page = build_draft_page(draft_title: "下書きタイトル", page_title: "公開タイトル")
      expect(draft_page.display_title).to eq("下書きタイトル")
    end

    it "下書きのタイトルがなくページのタイトルがある場合、ページのタイトルを返すこと" do
      draft_page = build_draft_page(draft_title: nil, page_title: "公開タイトル")
      expect(draft_page.display_title).to eq("公開タイトル")
    end

    it "どちらのタイトルもない場合、「無題」を返すこと" do
      draft_page = build_draft_page(draft_title: nil, page_title: nil)
      expect(draft_page.display_title).to eq(I18n.t("messages.pages.untitled"))
    end

    it "下書きのタイトルが空文字の場合、ページのタイトルにフォールバックすること" do
      draft_page = build_draft_page(draft_title: "", page_title: "公開タイトル")
      expect(draft_page.display_title).to eq("公開タイトル")
    end
  end

  private

  def build_draft_page(draft_title:, page_title:)
    space = Space.new(
      database_id: "space-id",
      identifier: "test",
      name: "Test",
      plan: Plan::Free,
      joined_at: Time.current,
      can_create_topic: nil
    )

    topic = Topic.new(
      database_id: "topic-id",
      number: 1,
      name: "Topic",
      description: "",
      visibility: TopicVisibility::Public,
      can_update: nil,
      can_create_page: nil,
      space:
    )

    page = Page.new(
      database_id: "page-id",
      number: 1,
      title: page_title,
      body: "",
      body_html: "",
      modified_at: Time.current,
      published_at: nil,
      pinned_at: nil,
      trashed_at: nil,
      can_update: nil,
      space:,
      topic:,
      card_image_url: nil,
      og_image_url: nil
    )

    DraftPage.new(
      database_id: "draft-id",
      title: draft_title,
      modified_at: Time.current,
      space:,
      page:
    )
  end
end
