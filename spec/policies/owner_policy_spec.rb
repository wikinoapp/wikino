# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe OwnerPolicy do
  it "Ownerロールで初期化できること" do
    user_record = FactoryBot.create(:user_record)
    space_record = FactoryBot.create(:space_record)
    space_member_record = FactoryBot.create(
      :space_member_record,
      user_record:,
      space_record:,
      role: SpaceMemberRole::Owner.serialize
    )

    policy = OwnerPolicy.new(user_record:, space_member_record:)

    expect(policy).to be_instance_of(OwnerPolicy)
  end

  describe "#can_update_space?" do
    it "同じスペースの場合はtrueを返すこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(
        :space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize
      )

      policy = OwnerPolicy.new(user_record:, space_member_record:)

      expect(policy.can_update_space?(space_record:)).to be(true)
    end

    it "異なるスペースの場合はfalseを返すこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      other_space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(
        :space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize
      )

      policy = OwnerPolicy.new(user_record:, space_member_record:)

      expect(policy.can_update_space?(space_record: other_space_record)).to be(false)
    end
  end

  describe "#can_update_topic?" do
    it "同じスペースのトピックなら参加していなくても編集可能であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(
        :space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize
      )
      topic_record = FactoryBot.create(:topic_record, space_record:)
      # TopicMemberRecordは作成しない（参加していない）

      policy = OwnerPolicy.new(user_record:, space_member_record:)

      expect(policy.can_update_topic?(topic_record:)).to be(true)
    end

    it "異なるスペースのトピックは編集不可であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      other_space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(
        :space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize
      )
      topic_record = FactoryBot.create(:topic_record, space_record: other_space_record)

      policy = OwnerPolicy.new(user_record:, space_member_record:)

      expect(policy.can_update_topic?(topic_record:)).to be(false)
    end
  end

  # AttachmentRecordのFactoryが複雑なため、attachment関連のテストは省略
  # 実際の動作は結合テストで確認

  describe "#can_manage_attachments?" do
    it "同じスペースなら管理画面にアクセス可能であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(
        :space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize
      )

      policy = OwnerPolicy.new(user_record:, space_member_record:)

      expect(policy.can_manage_attachments?(space_record:)).to be(true)
    end
  end

  describe "#can_export_space?" do
    it "同じスペースならエクスポート可能であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(
        :space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize
      )

      policy = OwnerPolicy.new(user_record:, space_member_record:)

      expect(policy.can_export_space?(space_record:)).to be(true)
    end

    it "異なるスペースならエクスポート不可であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      other_space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(
        :space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize
      )

      policy = OwnerPolicy.new(user_record:, space_member_record:)

      expect(policy.can_export_space?(space_record: other_space_record)).to be(false)
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
        role: SpaceMemberRole::Owner.serialize
      )

      policy = OwnerPolicy.new(user_record:, space_member_record:)

      expect(policy.can_create_topic?).to be(true)
    end
  end

  describe "#can_create_page?" do
    it "同じスペースのトピックならページ作成可能であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(
        :space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize
      )
      topic_record = FactoryBot.create(:topic_record, space_record:)

      policy = OwnerPolicy.new(user_record:, space_member_record:)

      expect(policy.can_create_page?(topic_record:)).to be(true)
    end

    it "異なるスペースのトピックならページ作成不可であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      other_space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(
        :space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize
      )
      topic_record = FactoryBot.create(:topic_record, space_record: other_space_record)

      policy = OwnerPolicy.new(user_record:, space_member_record:)

      expect(policy.can_create_page?(topic_record:)).to be(false)
    end
  end

  describe "#can_update_page?" do
    it "アクティブかつ同じスペースならページ編集可能であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(
        :space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize,
        active: true
      )
      topic_record = FactoryBot.create(:topic_record, space_record:)
      page_record = FactoryBot.create(:page_record, topic_record:, space_id: space_record.id)

      policy = OwnerPolicy.new(user_record:, space_member_record:)

      expect(policy.can_update_page?(page_record:)).to be(true)
    end

    it "非アクティブならページ編集不可であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(
        :space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize,
        active: false
      )
      topic_record = FactoryBot.create(:topic_record, space_record:)
      page_record = FactoryBot.create(:page_record, topic_record:, space_id: space_record.id)

      policy = OwnerPolicy.new(user_record:, space_member_record:)

      expect(policy.can_update_page?(page_record:)).to be(false)
    end
  end

  describe "#can_trash_page?" do
    it "アクティブかつ同じスペースならページ削除可能であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(
        :space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize,
        active: true
      )
      topic_record = FactoryBot.create(:topic_record, space_record:)
      page_record = FactoryBot.create(:page_record, topic_record:, space_id: space_record.id)

      policy = OwnerPolicy.new(user_record:, space_member_record:)

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
        role: SpaceMemberRole::Owner.serialize,
        active: true
      )

      policy = OwnerPolicy.new(user_record:, space_member_record:)

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
        role: SpaceMemberRole::Owner.serialize,
        active: true
      )

      policy = OwnerPolicy.new(user_record:, space_member_record:)

      expect(policy.can_create_bulk_restore_pages?).to be(true)
    end

    it "非アクティブなら一括復元不可であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      space_member_record = FactoryBot.create(
        :space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize,
        active: false
      )

      policy = OwnerPolicy.new(user_record:, space_member_record:)

      expect(policy.can_create_bulk_restore_pages?).to be(false)
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
        role: SpaceMemberRole::Owner.serialize,
        active: true
      )

      policy = OwnerPolicy.new(user_record:, space_member_record:)

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
        role: SpaceMemberRole::Owner.serialize
      )
      public_topic = FactoryBot.create(:topic_record, space_record:, visibility: TopicVisibility::Public.serialize)
      private_topic = FactoryBot.create(:topic_record, space_record:, visibility: TopicVisibility::Private.serialize)

      policy = OwnerPolicy.new(user_record:, space_member_record:)

      topics = policy.showable_topics(space_record:)
      expect(topics).to include(public_topic)
      expect(topics).to include(private_topic)
    end
  end
end
