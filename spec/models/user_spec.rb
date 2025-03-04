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
      create(:topic_member, :admin, space:, topic: topic_b, space_member:)
      create(:topic_member, :member, space:, topic: topic_c, space_member:)

      expect(user.viewable_topics).to contain_exactly(topic_a, topic_b, topic_c, topic_d)
    end
  end
end
