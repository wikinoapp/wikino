# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe TopicGuestPolicy do
  describe "#can_create_page?" do
    it "Topic Guestはページ作成が許可されないこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)

      # ユーザーがいない場合
      policy = TopicGuestPolicy.new(user_record: nil)
      expect(policy.can_create_page?(topic_record:)).to be(false)

      # ユーザーがいる場合でもページ作成不可
      policy = TopicGuestPolicy.new(user_record:)
      expect(policy.can_create_page?(topic_record:)).to be(false)
    end
  end

  describe "#can_update_topic?" do
    it "Topic Guestはトピックの基本情報を更新できないこと" do
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)

      # ユーザーがいない場合
      policy = TopicGuestPolicy.new(user_record: nil)
      expect(policy.can_update_topic?(topic_record:)).to be(false)

      # ユーザーがいる場合でも更新不可
      user_record = FactoryBot.create(:user_record)
      policy = TopicGuestPolicy.new(user_record:)
      expect(policy.can_update_topic?(topic_record:)).to be(false)
    end
  end

  describe "#can_delete_topic?" do
    it "Topic Guestはトピックを削除できないこと" do
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)

      # ユーザーがいない場合
      policy = TopicGuestPolicy.new(user_record: nil)
      expect(policy.can_delete_topic?(topic_record:)).to be(false)

      # ユーザーがいる場合でも削除不可
      user_record = FactoryBot.create(:user_record)
      policy = TopicGuestPolicy.new(user_record:)
      expect(policy.can_delete_topic?(topic_record:)).to be(false)
    end
  end

  describe "#can_manage_topic_members?" do
    it "Topic Guestはトピックメンバーを管理できないこと" do
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)

      # ユーザーがいない場合
      policy = TopicGuestPolicy.new(user_record: nil)
      expect(policy.can_manage_topic_members?(topic_record:)).to be(false)

      # ユーザーがいる場合でも管理不可
      user_record = FactoryBot.create(:user_record)
      policy = TopicGuestPolicy.new(user_record:)
      expect(policy.can_manage_topic_members?(topic_record:)).to be(false)
    end
  end

  describe "#can_update_page?" do
    it "Topic Guestはページを更新できないこと" do
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      page_record = FactoryBot.create(:page_record, topic_record:, space_record:)

      # ユーザーがいない場合
      policy = TopicGuestPolicy.new(user_record: nil)
      expect(policy.can_update_page?(page_record:)).to be(false)

      # ユーザーがいる場合でも更新不可
      user_record = FactoryBot.create(:user_record)
      policy = TopicGuestPolicy.new(user_record:)
      expect(policy.can_update_page?(page_record:)).to be(false)
    end
  end

  describe "#can_update_draft_page?" do
    it "Topic Guestはドラフトページを更新できないこと" do
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      page_record = FactoryBot.create(:page_record, topic_record:, space_record:)

      # ユーザーがいない場合
      policy = TopicGuestPolicy.new(user_record: nil)
      expect(policy.can_update_draft_page?(page_record:)).to be(false)

      # ユーザーがいる場合でも更新不可
      user_record = FactoryBot.create(:user_record)
      policy = TopicGuestPolicy.new(user_record:)
      expect(policy.can_update_draft_page?(page_record:)).to be(false)
    end
  end

  describe "#can_show_page?" do
    it "Topic Guestは公開トピックのページのみ閲覧可能であること" do
      space_record = FactoryBot.create(:space_record)

      # 公開トピックのページ
      public_topic_record = FactoryBot.create(:topic_record,
        space_record:,
        visibility: TopicVisibility::Public.serialize)
      public_page_record = FactoryBot.create(:page_record,
        topic_record: public_topic_record,
        space_record:)

      # 非公開トピックのページ
      private_topic_record = FactoryBot.create(:topic_record,
        space_record:,
        visibility: TopicVisibility::Private.serialize)
      private_page_record = FactoryBot.create(:page_record,
        topic_record: private_topic_record,
        space_record:)

      # ユーザーがいない場合
      policy = TopicGuestPolicy.new(user_record: nil)
      expect(policy.can_show_page?(page_record: public_page_record)).to be(true)
      expect(policy.can_show_page?(page_record: private_page_record)).to be(false)

      # ユーザーがいる場合でも同じ結果
      user_record = FactoryBot.create(:user_record)
      policy = TopicGuestPolicy.new(user_record:)
      expect(policy.can_show_page?(page_record: public_page_record)).to be(true)
      expect(policy.can_show_page?(page_record: private_page_record)).to be(false)
    end
  end

  describe "#can_trash_page?" do
    it "Topic Guestはページをゴミ箱に移動できないこと" do
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      page_record = FactoryBot.create(:page_record, topic_record:, space_record:)

      # ユーザーがいない場合
      policy = TopicGuestPolicy.new(user_record: nil)
      expect(policy.can_trash_page?(page_record:)).to be(false)

      # ユーザーがいる場合でも移動不可
      user_record = FactoryBot.create(:user_record)
      policy = TopicGuestPolicy.new(user_record:)
      expect(policy.can_trash_page?(page_record:)).to be(false)
    end
  end

  describe "#can_create_edit_suggestion?" do
    it "Topic Guestは編集提案を作成できないこと" do
      # ユーザーがいない場合
      policy = TopicGuestPolicy.new(user_record: nil)
      expect(policy.can_create_edit_suggestion?).to be false

      # ユーザーがいる場合でも作成不可
      user_record = FactoryBot.create(:user_record)
      policy = TopicGuestPolicy.new(user_record:)
      expect(policy.can_create_edit_suggestion?).to be false
    end
  end

  describe "#can_update_edit_suggestion?" do
    it "Topic Guestは編集提案を更新できないこと" do
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      user_record = FactoryBot.create(:user_record)
      space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:, active: true)
      edit_suggestion_record = FactoryBot.create(:edit_suggestion_record,
        space_record:,
        topic_record:,
        created_space_member_record: space_member_record)

      # ユーザーがいない場合
      policy = TopicGuestPolicy.new(user_record: nil)
      expect(policy.can_update_edit_suggestion?(edit_suggestion_record:)).to be false

      # ユーザーがいる場合でも更新不可
      policy = TopicGuestPolicy.new(user_record:)
      expect(policy.can_update_edit_suggestion?(edit_suggestion_record:)).to be false
    end
  end

  describe "#can_apply_edit_suggestion?" do
    it "Topic Guestは編集提案を反映できないこと" do
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      user_record = FactoryBot.create(:user_record)
      space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:, active: true)
      edit_suggestion_record = FactoryBot.create(:edit_suggestion_record,
        space_record:,
        topic_record:,
        created_space_member_record: space_member_record)

      # ユーザーがいない場合
      policy = TopicGuestPolicy.new(user_record: nil)
      expect(policy.can_apply_edit_suggestion?(edit_suggestion_record:)).to be false

      # ユーザーがいる場合でも反映不可
      policy = TopicGuestPolicy.new(user_record:)
      expect(policy.can_apply_edit_suggestion?(edit_suggestion_record:)).to be false
    end
  end

  describe "#can_close_edit_suggestion?" do
    it "Topic Guestは編集提案をクローズできないこと" do
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      user_record = FactoryBot.create(:user_record)
      space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:, active: true)
      edit_suggestion_record = FactoryBot.create(:edit_suggestion_record,
        space_record:,
        topic_record:,
        created_space_member_record: space_member_record)

      # ユーザーがいない場合
      policy = TopicGuestPolicy.new(user_record: nil)
      expect(policy.can_close_edit_suggestion?(edit_suggestion_record:)).to be false

      # ユーザーがいる場合でもクローズ不可
      policy = TopicGuestPolicy.new(user_record:)
      expect(policy.can_close_edit_suggestion?(edit_suggestion_record:)).to be false
    end
  end

  describe "#can_comment_on_edit_suggestion?" do
    it "Topic Guestは編集提案にコメントできないこと" do
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      user_record = FactoryBot.create(:user_record)
      space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:, active: true)
      edit_suggestion_record = FactoryBot.create(:edit_suggestion_record,
        space_record:,
        topic_record:,
        created_space_member_record: space_member_record)

      # ユーザーがいない場合
      policy = TopicGuestPolicy.new(user_record: nil)
      expect(policy.can_comment_on_edit_suggestion?(edit_suggestion_record:)).to be false

      # ユーザーがいる場合でもコメント不可
      policy = TopicGuestPolicy.new(user_record:)
      expect(policy.can_comment_on_edit_suggestion?(edit_suggestion_record:)).to be false
    end
  end
end
