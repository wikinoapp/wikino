# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe SpaceMemberPolicy do
  it "Memberロールで初期化できること" do
    user_record = FactoryBot.create(:user_record)
    space_record = FactoryBot.create(:space_record)
    space_member_record = FactoryBot.create(
      :space_member_record,
      user_record:,
      space_record:,
      role: SpaceMemberRole::Member.serialize
    )

    policy = SpaceMemberPolicy.new(user_record:, space_member_record:)

    expect(policy).to be_instance_of(SpaceMemberPolicy)
  end

  describe "#can_update_space?" do
    it "常にfalseを返すこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(
        :space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize
      )

      policy = SpaceMemberPolicy.new(user_record:, space_member_record:)

      expect(policy.can_update_space?(space_record:)).to be(false)
    end
  end

  describe "#can_update_topic?" do
    it "参加しているトピックは編集可能であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(
        :space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize
      )
      topic_record = FactoryBot.create(:topic_record, space_record:)
      FactoryBot.create(:topic_member_record, space_member_record:, topic_record:)

      policy = SpaceMemberPolicy.new(user_record:, space_member_record:)

      expect(policy.can_update_topic?(topic_record:)).to be(true)
    end

    it "参加していないトピックは編集不可であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(
        :space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize
      )
      topic_record = FactoryBot.create(:topic_record, space_record:)
      # TopicMemberRecordは作成しない

      policy = SpaceMemberPolicy.new(user_record:, space_member_record:)

      expect(policy.can_update_topic?(topic_record:)).to be(false)
    end
  end

  describe "#can_delete_topic?" do
    it "常にfalseを返すこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(
        :space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize
      )
      topic_record = FactoryBot.create(:topic_record, space_record:)

      policy = SpaceMemberPolicy.new(user_record:, space_member_record:)

      expect(policy.can_delete_topic?(topic_record:)).to be(false)
    end
  end

  describe "#can_manage_topic_members?" do
    it "常にfalseを返すこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(
        :space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize
      )
      topic_record = FactoryBot.create(:topic_record, space_record:)

      policy = SpaceMemberPolicy.new(user_record:, space_member_record:)

      expect(policy.can_manage_topic_members?(topic_record:)).to be(false)
    end
  end

  # AttachmentRecordのFactoryが複雑なため、attachment関連のテストは省略
  # 実際の動作は結合テストで確認

  describe "#can_manage_attachments?" do
    it "常にfalseを返すこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(
        :space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize
      )

      policy = SpaceMemberPolicy.new(user_record:, space_member_record:)

      expect(policy.can_manage_attachments?(space_record:)).to be(false)
    end
  end

  describe "#can_export_space?" do
    it "常にfalseを返すこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(
        :space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize
      )

      policy = SpaceMemberPolicy.new(user_record:, space_member_record:)

      expect(policy.can_export_space?(space_record:)).to be(false)
    end
  end

  describe "#can_create_topic?" do
    it "スペースに参加していればトピック作成可能であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(
        :space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize
      )

      policy = SpaceMemberPolicy.new(user_record:, space_member_record:)

      expect(policy.can_create_topic?).to be(true)
    end
  end

  describe "#can_create_page?" do
    it "参加しているトピックならページ作成可能であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(
        :space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize
      )
      topic_record = FactoryBot.create(:topic_record, space_record:)
      FactoryBot.create(:topic_member_record, space_member_record:, topic_record:)

      policy = SpaceMemberPolicy.new(user_record:, space_member_record:)

      expect(policy.can_create_page?(topic_record:)).to be(true)
    end

    it "参加していないトピックならページ作成不可であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(
        :space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize
      )
      topic_record = FactoryBot.create(:topic_record, space_record:)

      policy = SpaceMemberPolicy.new(user_record:, space_member_record:)

      expect(policy.can_create_page?(topic_record:)).to be(false)
    end
  end

  describe "#can_update_page?" do
    it "参加しているトピックのページは編集可能であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(
        :space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize
      )
      topic_record = FactoryBot.create(:topic_record, space_record:)
      FactoryBot.create(:topic_member_record, space_member_record:, topic_record:)
      page_record = FactoryBot.create(:page_record, space_record:, topic_record:)

      policy = SpaceMemberPolicy.new(user_record:, space_member_record:)

      expect(policy.can_update_page?(page_record:)).to be(true)
    end

    it "参加していないトピックのページは編集不可であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(
        :space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize
      )
      topic_record = FactoryBot.create(:topic_record, space_record:)
      # TopicMemberRecordは作成しない
      page_record = FactoryBot.create(:page_record, space_record:, topic_record:)

      policy = SpaceMemberPolicy.new(user_record:, space_member_record:)

      expect(policy.can_update_page?(page_record:)).to be(false)
    end
  end

  describe "#can_trash_page?" do
    it "参加しているトピックのページは削除可能であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(
        :space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize
      )
      topic_record = FactoryBot.create(:topic_record, space_record:)
      FactoryBot.create(:topic_member_record, space_member_record:, topic_record:)
      page_record = FactoryBot.create(:page_record, space_record:, topic_record:)

      policy = SpaceMemberPolicy.new(user_record:, space_member_record:)

      expect(policy.can_trash_page?(page_record:)).to be(true)
    end
  end

  describe "#can_show_trash?" do
    it "アクティブかつ同じスペースならゴミ箱閲覧可能であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(
        :space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize,
        active: true
      )

      policy = SpaceMemberPolicy.new(user_record:, space_member_record:)

      expect(policy.can_show_trash?(space_record:)).to be(true)
    end
  end

  describe "#can_create_bulk_restore_pages?" do
    it "アクティブなら一括復元可能であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(
        :space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize,
        active: true
      )

      policy = SpaceMemberPolicy.new(user_record:, space_member_record:)

      expect(policy.can_create_bulk_restore_pages?).to be(true)
    end
  end

  describe "#can_upload_attachment?" do
    it "アクティブかつ同じスペースならファイルアップロード可能であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(
        :space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize,
        active: true
      )

      policy = SpaceMemberPolicy.new(user_record:, space_member_record:)

      expect(policy.can_upload_attachment?(space_record:)).to be(true)
    end
  end

  describe "#showable_topics" do
    it "全てのトピックが閲覧可能であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(
        :space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize
      )
      public_topic = FactoryBot.create(:topic_record, space_record:, visibility: TopicVisibility::Public.serialize)
      private_topic = FactoryBot.create(:topic_record, space_record:, visibility: TopicVisibility::Private.serialize)

      policy = SpaceMemberPolicy.new(user_record:, space_member_record:)

      topics = policy.showable_topics(space_record:)
      expect(topics).to include(public_topic)
      expect(topics).to include(private_topic)
    end
  end
end
