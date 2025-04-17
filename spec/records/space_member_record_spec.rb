# typed: false
# frozen_string_literal: true

RSpec.describe SpaceMemberRecord, type: :record do
  describe "#last_modified_pages" do
    it "ページが存在するとき、最後に編集したページから取得できること" do
      user_a = create(:user_record)
      user_b = create(:user_record)
      space = create(:space_record)
      space_member_a = create(:space_member_record, space_record: space, user_record: user_a)
      space_member_b = create(:space_member_record, space_record: space, user_record: user_b)

      topic = create(:topic_record, space_record: space)
      page_a = create(:page_record, space_record: space, topic_record: topic, title: "ページA")
      page_b = create(:page_record, space_record: space, topic_record: topic, title: "ページB")
      page_c = create(:page_record, space_record: space, topic_record: topic, title: "ページC")

      create(:page_editor_record, space_record: space, page_record: page_a, space_member_record: space_member_a, last_page_modified_at: Time.zone.parse("2024-08-18 0:00:00"))
      create(:page_editor_record, space_record: space, page_record: page_b, space_member_record: space_member_b, last_page_modified_at: Time.zone.parse("2024-08-18 1:00:00"))
      create(:page_editor_record, space_record: space, page_record: page_c, space_member_record: space_member_b, last_page_modified_at: Time.zone.parse("2024-08-18 2:00:00"))
      create(:page_editor_record, space_record: space, page_record: page_c, space_member_record: space_member_a, last_page_modified_at: Time.zone.parse("2024-08-18 3:00:00"))

      expect(space_member_a.last_modified_pages.pluck(:title)).to eq(%w[ページC ページA])
    end
  end
end
