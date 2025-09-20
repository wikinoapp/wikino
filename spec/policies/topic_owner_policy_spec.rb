# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe TopicOwnerPolicy do
  describe "#initialize" do
    it "user_recordとspace_member_recordのユーザーが一致しない場合はエラーになること" do
      user_record = FactoryBot.create(:user_record)
      other_user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record: other_user_record,
        space_record:,
        role: SpaceMemberRole::Owner.serialize)

      expect {
        TopicOwnerPolicy.new(
          user_record:,
          space_member_record:
        )
      }.to raise_error(ArgumentError, /Mismatched relations/)
    end
  end

  describe "#can_update_topic?" do
    it "Topic Ownerは同じスペースのトピックの基本情報を更新可能であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize)

      policy = TopicOwnerPolicy.new(
        user_record:,
        space_member_record:
      )

      expect(policy.can_update_topic?(topic_record:)).to be(true)
    end

    it "Topic Ownerは別のスペースのトピックの基本情報を更新できないこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      other_space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record: other_space_record)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize)

      policy = TopicOwnerPolicy.new(
        user_record:,
        space_member_record:
      )

      expect(policy.can_update_topic?(topic_record:)).to be(false)
    end

    it "非アクティブな場合はトピックの基本情報を更新できないこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize,
        active: false)

      policy = TopicOwnerPolicy.new(
        user_record:,
        space_member_record:
      )

      expect(policy.can_update_topic?(topic_record:)).to be(false)
    end
  end

  describe "#can_delete_topic?" do
    it "Topic Ownerは同じスペースのトピックを削除可能であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize)

      policy = TopicOwnerPolicy.new(
        user_record:,
        space_member_record:
      )

      expect(policy.can_delete_topic?(topic_record:)).to be(true)
    end

    it "Topic Ownerは別のスペースのトピックを削除できないこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      other_space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record: other_space_record)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize)

      policy = TopicOwnerPolicy.new(
        user_record:,
        space_member_record:
      )

      expect(policy.can_delete_topic?(topic_record:)).to be(false)
    end

    it "非アクティブな場合はトピックを削除できないこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize,
        active: false)

      policy = TopicOwnerPolicy.new(
        user_record:,
        space_member_record:
      )

      expect(policy.can_delete_topic?(topic_record:)).to be(false)
    end
  end

  describe "#can_manage_topic_members?" do
    it "Topic Ownerは同じスペースのトピックメンバーを管理可能であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize)

      policy = TopicOwnerPolicy.new(
        user_record:,
        space_member_record:
      )

      expect(policy.can_manage_topic_members?(topic_record:)).to be(true)
    end

    it "Topic Ownerは別のスペースのトピックメンバーを管理できないこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      other_space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record: other_space_record)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize)

      policy = TopicOwnerPolicy.new(
        user_record:,
        space_member_record:
      )

      expect(policy.can_manage_topic_members?(topic_record:)).to be(false)
    end

    it "非アクティブな場合はトピックメンバーを管理できないこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize,
        active: false)

      policy = TopicOwnerPolicy.new(
        user_record:,
        space_member_record:
      )

      expect(policy.can_manage_topic_members?(topic_record:)).to be(false)
    end
  end

  describe "#can_create_page?" do
    it "Topic Ownerは同じスペースのトピックにページを作成可能であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize)

      policy = TopicOwnerPolicy.new(
        user_record:,
        space_member_record:
      )

      expect(policy.can_create_page?(topic_record:)).to be(true)
    end

    it "Topic Ownerは別のスペースのトピックにページを作成できないこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      other_space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record: other_space_record)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize)

      policy = TopicOwnerPolicy.new(
        user_record:,
        space_member_record:
      )

      expect(policy.can_create_page?(topic_record:)).to be(false)
    end

    it "非アクティブな場合はページを作成できないこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize,
        active: false)

      policy = TopicOwnerPolicy.new(
        user_record:,
        space_member_record:
      )

      expect(policy.can_create_page?(topic_record:)).to be(false)
    end
  end

  describe "#can_update_page?" do
    it "Topic Ownerは同じスペースのページを更新可能であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      page_record = FactoryBot.create(:page_record, topic_record:, space_record:)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize)

      policy = TopicOwnerPolicy.new(
        user_record:,
        space_member_record:
      )

      expect(policy.can_update_page?(page_record:)).to be(true)
    end

    it "Topic Ownerは別のスペースのページを更新できないこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      other_space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record: other_space_record)
      page_record = FactoryBot.create(:page_record, topic_record:, space_record: other_space_record)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize)

      policy = TopicOwnerPolicy.new(
        user_record:,
        space_member_record:
      )

      expect(policy.can_update_page?(page_record:)).to be(false)
    end

    it "非アクティブな場合はページを更新できないこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      page_record = FactoryBot.create(:page_record, topic_record:, space_record:)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize,
        active: false)

      policy = TopicOwnerPolicy.new(
        user_record:,
        space_member_record:
      )

      expect(policy.can_update_page?(page_record:)).to be(false)
    end
  end

  describe "#can_update_draft_page?" do
    it "can_update_page?と同じ結果を返すこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      page_record = FactoryBot.create(:page_record, topic_record:, space_record:)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize)

      policy = TopicOwnerPolicy.new(
        user_record:,
        space_member_record:
      )

      expect(policy.can_update_draft_page?(page_record:)).to eq(policy.can_update_page?(page_record:))
    end
  end

  describe "#can_show_page?" do
    it "公開トピックのページは誰でも閲覧可能であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      other_space_record = FactoryBot.create(:space_record)

      public_topic_record = FactoryBot.create(:topic_record,
        space_record: other_space_record,
        visibility: TopicVisibility::Public.serialize)
      public_page_record = FactoryBot.create(:page_record,
        topic_record: public_topic_record,
        space_record: other_space_record)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize)

      policy = TopicOwnerPolicy.new(
        user_record:,
        space_member_record:
      )

      # 別のスペースの公開トピックのページも閲覧可能
      expect(policy.can_show_page?(page_record: public_page_record)).to be(true)
    end

    it "非公開トピックのページは同じスペースのメンバーのみ閲覧可能であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      other_space_record = FactoryBot.create(:space_record)

      # 同じスペースの非公開トピック
      private_topic_record = FactoryBot.create(:topic_record,
        space_record:,
        visibility: TopicVisibility::Private.serialize)
      private_page_record = FactoryBot.create(:page_record,
        topic_record: private_topic_record,
        space_record:)

      # 別のスペースの非公開トピック
      other_private_topic_record = FactoryBot.create(:topic_record,
        space_record: other_space_record,
        visibility: TopicVisibility::Private.serialize)
      other_private_page_record = FactoryBot.create(:page_record,
        topic_record: other_private_topic_record,
        space_record: other_space_record)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize)

      policy = TopicOwnerPolicy.new(
        user_record:,
        space_member_record:
      )

      # 同じスペースの非公開トピックのページは閲覧可能
      expect(policy.can_show_page?(page_record: private_page_record)).to be(true)

      # 別のスペースの非公開トピックのページは閲覧不可
      expect(policy.can_show_page?(page_record: other_private_page_record)).to be(false)
    end

    it "非アクティブな場合は非公開トピックのページを閲覧できないこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)

      private_topic_record = FactoryBot.create(:topic_record,
        space_record:,
        visibility: TopicVisibility::Private.serialize)
      private_page_record = FactoryBot.create(:page_record,
        topic_record: private_topic_record,
        space_record:)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize,
        active: false)

      policy = TopicOwnerPolicy.new(
        user_record:,
        space_member_record:
      )

      expect(policy.can_show_page?(page_record: private_page_record)).to be(false)
    end
  end

  describe "#can_trash_page?" do
    it "Topic Ownerは同じスペースのページをゴミ箱に移動可能であること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      page_record = FactoryBot.create(:page_record, topic_record:, space_record:)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize)

      policy = TopicOwnerPolicy.new(
        user_record:,
        space_member_record:
      )

      expect(policy.can_trash_page?(page_record:)).to be(true)
    end

    it "Topic Ownerは別のスペースのページをゴミ箱に移動できないこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      other_space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record: other_space_record)
      page_record = FactoryBot.create(:page_record, topic_record:, space_record: other_space_record)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize)

      policy = TopicOwnerPolicy.new(
        user_record:,
        space_member_record:
      )

      expect(policy.can_trash_page?(page_record:)).to be(false)
    end

    it "非アクティブな場合はページをゴミ箱に移動できないこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      page_record = FactoryBot.create(:page_record, topic_record:, space_record:)

      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize,
        active: false)

      policy = TopicOwnerPolicy.new(
        user_record:,
        space_member_record:
      )

      expect(policy.can_trash_page?(page_record:)).to be(false)
    end
  end

  describe "#can_create_edit_suggestion?" do
    it "Topic Ownerは編集提案を作成できること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:, active: true, role: SpaceMemberRole::Owner.serialize)

      policy = TopicOwnerPolicy.new(user_record:, space_member_record:)

      expect(policy.can_create_edit_suggestion?).to be true
    end

    it "非アクティブなOwnerは編集提案を作成できないこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:, active: false, role: SpaceMemberRole::Owner.serialize)

      policy = TopicOwnerPolicy.new(user_record:, space_member_record:)

      expect(policy.can_create_edit_suggestion?).to be false
    end
  end

  describe "#can_update_edit_suggestion?" do
    it "作成者は自分の編集提案を更新できること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:, active: true, role: SpaceMemberRole::Owner.serialize)
      edit_suggestion_record = FactoryBot.create(:edit_suggestion_record,
        space_record:,
        topic_record:,
        created_user_record: user_record)

      policy = TopicOwnerPolicy.new(user_record:, space_member_record:)

      expect(policy.can_update_edit_suggestion?(edit_suggestion_record:)).to be true
    end

    it "他のユーザーの編集提案は更新できないこと" do
      user_record = FactoryBot.create(:user_record)
      other_user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:, active: true, role: SpaceMemberRole::Owner.serialize)
      edit_suggestion_record = FactoryBot.create(:edit_suggestion_record,
        space_record:,
        topic_record:,
        created_user_record: other_user_record)

      policy = TopicOwnerPolicy.new(user_record:, space_member_record:)

      expect(policy.can_update_edit_suggestion?(edit_suggestion_record:)).to be false
    end
  end

  describe "#can_apply_edit_suggestion?" do
    it "Topic Ownerは同じスペースの編集提案を反映できること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:, active: true, role: SpaceMemberRole::Owner.serialize)
      edit_suggestion_record = FactoryBot.create(:edit_suggestion_record,
        space_record:,
        topic_record:,
        created_user_record: user_record)

      policy = TopicOwnerPolicy.new(user_record:, space_member_record:)

      expect(policy.can_apply_edit_suggestion?(edit_suggestion_record:)).to be true
    end

    it "Topic Ownerは他のスペースの編集提案を反映できないこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      other_space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record: other_space_record)
      space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:, active: true, role: SpaceMemberRole::Owner.serialize)
      edit_suggestion_record = FactoryBot.create(:edit_suggestion_record,
        space_record: other_space_record,
        topic_record:,
        created_user_record: user_record)

      policy = TopicOwnerPolicy.new(user_record:, space_member_record:)

      expect(policy.can_apply_edit_suggestion?(edit_suggestion_record:)).to be false
    end
  end

  describe "#can_close_edit_suggestion?" do
    it "Topic Ownerは同じスペースの編集提案をクローズできること" do
      user_record = FactoryBot.create(:user_record)
      other_user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:, active: true, role: SpaceMemberRole::Owner.serialize)
      edit_suggestion_record = FactoryBot.create(:edit_suggestion_record,
        space_record:,
        topic_record:,
        created_user_record: other_user_record)

      policy = TopicOwnerPolicy.new(user_record:, space_member_record:)

      expect(policy.can_close_edit_suggestion?(edit_suggestion_record:)).to be true
    end
  end

  describe "#can_comment_on_edit_suggestion?" do
    it "Topic Ownerは同じスペースの編集提案にコメントできること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:, active: true, role: SpaceMemberRole::Owner.serialize)
      edit_suggestion_record = FactoryBot.create(:edit_suggestion_record,
        space_record:,
        topic_record:,
        created_user_record: user_record)

      policy = TopicOwnerPolicy.new(user_record:, space_member_record:)

      expect(policy.can_comment_on_edit_suggestion?(edit_suggestion_record:)).to be true
    end
  end
end
