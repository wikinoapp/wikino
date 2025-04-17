# typed: false
# frozen_string_literal: true

RSpec.describe UserRecord, type: :record do
  describe "#viewable_topics" do
    it "トピックが存在するとき、閲覧可能なトピックを返すこと" do
      user = create(:user_record)
      space = create(:space_record)
      space_member = create(:space_member_record, space_record: space, user_record: user)

      topic_a = create(:topic_record, :public, space_record: space, name: "トピックA")
      topic_b = create(:topic_record, :private, space_record: space, name: "トピックB")
      topic_c = create(:topic_record, :private, space_record: space, name: "トピックC")
      topic_d = create(:topic_record, :private, space_record: space, name: "トピックD")
      create(:topic_member_record, :admin, space_record: space, topic_record: topic_b, space_member_record: space_member)
      create(:topic_member_record, :member, space_record: space, topic_record: topic_c, space_member_record: space_member)

      expect(user.viewable_topics).to contain_exactly(topic_a, topic_b, topic_c, topic_d)
    end
  end
end
