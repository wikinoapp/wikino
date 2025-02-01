# typed: false
# frozen_string_literal: true

RSpec.describe User, type: :model do
  describe "#viewable_topics" do
    it "トピックが存在するとき、閲覧可能なトピックを返すこと" do
      user = create(:user)
      space = create(:space)
      space_member = create(:space_member, space:, user:)

      topic_a = create(:topic, :public, space:, name: "トピックA")
      topic_b = create(:topic, :private, space:, name: "トピックB")
      topic_c = create(:topic, :private, space:, name: "トピックC")
      topic_d = create(:topic, :private, space:, name: "トピックD")
      create(:topic_membership, :admin, space:, topic: topic_b, member: space_member)
      create(:topic_membership, :member, space:, topic: topic_c, member: space_member)

      expect(user.viewable_topics).to contain_exactly(topic_a, topic_b, topic_c, topic_d)
    end
  end

  describe "#last_page_modified_topics" do
    it "トピックが存在するとき、ページが編集された順にトピックが取得できること" do
      user = create(:user)
      space = create(:space)
      space_member = create(:space_member, space:, user:)

      topic_a = create(:topic, space:, name: "トピックA")
      topic_b = create(:topic, space:, name: "トピックB")
      topic_c = create(:topic, space:, name: "トピックC")
      topic_d = create(:topic, space:, name: "トピックD")
      create(:topic, space:, name: "トピックE")
      create(:topic_membership, space:, topic: topic_a, member: space_member, joined_at: Time.zone.parse("2024-08-18 0:00:00"), last_page_modified_at: nil)
      create(:topic_membership, space:, topic: topic_b, member: space_member, joined_at: Time.zone.parse("2024-08-18 1:00:00"), last_page_modified_at: Time.zone.parse("2024-08-19 0:00:00"))
      create(:topic_membership, space:, topic: topic_c, member: space_member, joined_at: Time.zone.parse("2024-08-18 2:00:00"), last_page_modified_at: Time.zone.parse("2024-08-19 1:00:00"))
      create(:topic_membership, space:, topic: topic_d, member: space_member, joined_at: Time.zone.parse("2024-08-18 3:00:00"), last_page_modified_at: nil)

      expect(
        user.last_page_modified_topics.pluck(:name)
      ).to eq(%w[トピックC トピックB トピックD トピックA])
    end
  end
end
