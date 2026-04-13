# typed: false
# frozen_string_literal: true

RSpec.describe TopicMemberRecord, type: :record do
  describe "#scopes" do
    it "scopesを読み取れること" do
      space_record = FactoryBot.create(:space_record)
      user_record = FactoryBot.create(:user_record)
      space_member_record = FactoryBot.create(
        :space_member_record,
        space_record:,
        user_record:
      )
      topic_record = FactoryBot.create(
        :topic_record,
        space_record:
      )
      topic_member_record = FactoryBot.create(
        :topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:,
        scopes: ["topic:read"]
      )

      expect(topic_member_record.reload.scopes).to eq(["topic:read"])
    end

    it "空配列がデフォルトであること" do
      space_record = FactoryBot.create(:space_record)
      user_record = FactoryBot.create(:user_record)
      space_member_record = FactoryBot.create(
        :space_member_record,
        space_record:,
        user_record:
      )
      topic_record = FactoryBot.create(
        :topic_record,
        space_record:
      )
      topic_member_record = FactoryBot.create(
        :topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:
      )

      expect(topic_member_record.reload.scopes).to eq([])
    end

    it "複数のスコープを保存・取得できること" do
      space_record = FactoryBot.create(:space_record)
      user_record = FactoryBot.create(:user_record)
      space_member_record = FactoryBot.create(
        :space_member_record,
        space_record:,
        user_record:
      )
      topic_record = FactoryBot.create(
        :topic_record,
        space_record:
      )
      topic_member_record = FactoryBot.create(
        :topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:,
        scopes: ["topic:read", "page:write", "suggestion:write"]
      )

      expect(topic_member_record.reload.scopes).to eq(
        ["topic:read", "page:write", "suggestion:write"]
      )
    end
  end
end
