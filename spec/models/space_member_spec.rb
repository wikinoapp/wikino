# typed: false
# frozen_string_literal: true

RSpec.describe SpaceMember, type: :model do
  describe "#last_modified_pages" do
    it "ページが存在するとき、最後に編集したページから取得できること" do
      user_a = create(:user)
      user_b = create(:user)
      space = create(:space)
      space_member_a = create(:space_member, space:, user: user_a)
      space_member_b = create(:space_member, space:, user: user_b)

      topic = create(:topic, space:)
      page_a = create(:page, space:, topic:, title: "ページA")
      page_b = create(:page, space:, topic:, title: "ページB")
      page_c = create(:page, space:, topic:, title: "ページC")

      create(:page_editorship, space:, page: page_a, editor: space_member_a, last_page_modified_at: Time.zone.parse("2024-08-18 0:00:00"))
      create(:page_editorship, space:, page: page_b, editor: space_member_b, last_page_modified_at: Time.zone.parse("2024-08-18 1:00:00"))
      create(:page_editorship, space:, page: page_c, editor: space_member_b, last_page_modified_at: Time.zone.parse("2024-08-18 2:00:00"))
      create(:page_editorship, space:, page: page_c, editor: space_member_a, last_page_modified_at: Time.zone.parse("2024-08-18 3:00:00"))

      expect(space_member_a.last_modified_pages.pluck(:title)).to eq(%w[ページC ページA])
    end
  end
end
