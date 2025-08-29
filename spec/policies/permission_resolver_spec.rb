# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe PermissionResolver do
  describe "#resolve" do
    it "Space Ownerの場合、OwnerPolicyを返すこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize)

      resolver = PermissionResolver.new(
        user_record:,
        space_record:,
        topic_record: nil
      )

      policy = resolver.resolve

      expect(policy).to be_a(SpaceOwnerPolicy)
    end

    it "Space Memberの場合、MemberPolicyを返すこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize)

      resolver = PermissionResolver.new(
        user_record:,
        space_record:,
        topic_record: nil
      )

      policy = resolver.resolve

      expect(policy).to be_a(SpaceMemberPolicy)
    end

    it "非メンバーの場合、GuestPolicyを返すこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)

      resolver = PermissionResolver.new(
        user_record:,
        space_record:,
        topic_record: nil
      )

      policy = resolver.resolve

      expect(policy).to be_a(SpaceGuestPolicy)
    end

    it "ユーザーがnilの場合、GuestPolicyを返すこと" do
      space_record = FactoryBot.create(:space_record)

      resolver = PermissionResolver.new(
        user_record: nil,
        space_record:,
        topic_record: nil
      )

      policy = resolver.resolve

      expect(policy).to be_a(SpaceGuestPolicy)
    end

    it "Topic Adminの場合、OwnerPolicyを返すこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize)
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:,
        role: TopicMemberRole::Admin.serialize)

      resolver = PermissionResolver.new(
        user_record:,
        space_record:,
        topic_record:
      )

      policy = resolver.resolve

      expect(policy).to be_a(SpaceOwnerPolicy)
    end

    it "Topic Memberの場合、MemberPolicyを返すこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize)
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:,
        role: TopicMemberRole::Member.serialize)

      resolver = PermissionResolver.new(
        user_record:,
        space_record:,
        topic_record:
      )

      policy = resolver.resolve

      expect(policy).to be_a(SpaceMemberPolicy)
    end

    it "Space OwnerはTopic権限よりも優先されること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize)
      # Topic MemberであってもSpace Ownerが優先される
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:,
        role: TopicMemberRole::Member.serialize)

      resolver = PermissionResolver.new(
        user_record:,
        space_record:,
        topic_record:
      )

      policy = resolver.resolve

      # Space Ownerの権限が優先される
      expect(policy).to be_a(SpaceOwnerPolicy)
    end
  end

  describe "#resolve_for_topic" do
    it "topic_recordがnilの場合、通常のresolveと同じ結果を返すこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize)

      resolver = PermissionResolver.new(
        user_record:,
        space_record:,
        topic_record: nil
      )

      policy = resolver.resolve_for_topic

      expect(policy).to be_a(SpaceMemberPolicy)
    end

    it "topic_recordが指定されている場合、Topic権限を考慮すること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)
      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize)
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:,
        role: TopicMemberRole::Admin.serialize)

      resolver = PermissionResolver.new(
        user_record:,
        space_record:,
        topic_record:
      )

      policy = resolver.resolve_for_topic

      expect(policy).to be_a(SpaceOwnerPolicy)
    end
  end

  describe "権限の優先順位" do
    it "Space Owner > Topic Admin > Topic Member > Space Member > Guestの順で優先されること" do
      # テストデータの準備
      space_owner_record = FactoryBot.create(:user_record)
      topic_admin_record = FactoryBot.create(:user_record)
      topic_member_record = FactoryBot.create(:user_record)
      space_member_record = FactoryBot.create(:user_record)
      guest_record = FactoryBot.create(:user_record)

      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)

      # Space Owner
      FactoryBot.create(:space_member_record,
        user_record: space_owner_record,
        space_record:,
        role: SpaceMemberRole::Owner.serialize)

      # Topic Admin (Space Memberでもある)
      topic_admin_space_member = FactoryBot.create(:space_member_record,
        user_record: topic_admin_record,
        space_record:,
        role: SpaceMemberRole::Member.serialize)
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record: topic_admin_space_member,
        role: TopicMemberRole::Admin.serialize)

      # Topic Member (Space Memberでもある)
      topic_member_space_member = FactoryBot.create(:space_member_record,
        user_record: topic_member_record,
        space_record:,
        role: SpaceMemberRole::Member.serialize)
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record: topic_member_space_member,
        role: TopicMemberRole::Member.serialize)

      # Space Member (Topicには参加していない)
      FactoryBot.create(:space_member_record,
        user_record: space_member_record,
        space_record:,
        role: SpaceMemberRole::Member.serialize)

      # Guest (どこにも所属していない)
      # guest_recordは何も追加の設定なし

      # 各ユーザーの権限を確認
      # 1. Space Owner
      resolver = PermissionResolver.new(
        user_record: space_owner_record,
        space_record:,
        topic_record:
      )
      expect(resolver.resolve).to be_a(SpaceOwnerPolicy)

      # 2. Topic Admin
      resolver = PermissionResolver.new(
        user_record: topic_admin_record,
        space_record:,
        topic_record:
      )
      expect(resolver.resolve).to be_a(SpaceOwnerPolicy) # Topic AdminもOwnerPolicyを持つ

      # 3. Topic Member
      resolver = PermissionResolver.new(
        user_record: topic_member_record,
        space_record:,
        topic_record:
      )
      expect(resolver.resolve).to be_a(SpaceMemberPolicy)

      # 4. Space Member (Topicに参加していない)
      resolver = PermissionResolver.new(
        user_record: space_member_record,
        space_record:,
        topic_record: # Topicを指定してもTopicMemberではない
      )
      expect(resolver.resolve).to be_a(SpaceMemberPolicy)

      # 5. Guest
      resolver = PermissionResolver.new(
        user_record: guest_record,
        space_record:,
        topic_record:
      )
      expect(resolver.resolve).to be_a(SpaceGuestPolicy)
    end

    it "Space OwnerがTopic Memberであっても、Space Ownerの権限が優先されること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)

      # Space Ownerとして設定
      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize)

      # Topic Memberとしても追加（通常のメンバー）
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:,
        role: TopicMemberRole::Member.serialize)

      resolver = PermissionResolver.new(
        user_record:,
        space_record:,
        topic_record:
      )

      # Space Ownerの権限が優先される
      policy = resolver.resolve
      expect(policy).to be_a(SpaceOwnerPolicy)
    end
  end

  describe "Topic指定有無による挙動" do
    it "Topicが指定されていない場合、TopicMemberRecordは考慮されないこと" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)

      # Space Memberとして設定
      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize)

      # Topic Adminとして設定
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:,
        role: TopicMemberRole::Admin.serialize)

      # Topicを指定しない場合
      resolver = PermissionResolver.new(
        user_record:,
        space_record:,
        topic_record: nil # Topicを指定しない
      )

      # Topic Adminであっても、Topic指定がなければSpace Memberとして扱われる
      policy = resolver.resolve
      expect(policy).to be_a(SpaceMemberPolicy)
    end

    it "Topicが指定されている場合、TopicMemberRecordが考慮されること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record:)

      # Space Memberとして設定
      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize)

      # Topic Adminとして設定
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record:,
        space_member_record:,
        role: TopicMemberRole::Admin.serialize)

      # Topicを指定する場合
      resolver = PermissionResolver.new(
        user_record:,
        space_record:,
        topic_record: # Topicを指定
      )

      # Topic Adminの権限が適用される
      policy = resolver.resolve
      expect(policy).to be_a(SpaceOwnerPolicy)
    end

    it "異なるTopicを指定した場合、そのTopicのメンバーでなければSpace権限が適用されること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)
      topic_record1 = FactoryBot.create(:topic_record, space_record:)
      topic_record2 = FactoryBot.create(:topic_record, space_record:)

      # Space Memberとして設定
      space_member_record = FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Member.serialize)

      # topic1のAdminとして設定
      FactoryBot.create(:topic_member_record,
        space_record:,
        topic_record: topic_record1,
        space_member_record:,
        role: TopicMemberRole::Admin.serialize)

      # topic2を指定してresolve（topic2のメンバーではない）
      resolver = PermissionResolver.new(
        user_record:,
        space_record:,
        topic_record: topic_record2
      )

      # topic2のメンバーではないので、Space Memberの権限が適用される
      policy = resolver.resolve
      expect(policy).to be_a(SpaceMemberPolicy)
    end
  end

  describe "エッジケース" do
    it "space_recordがnilの場合でもエラーにならないこと" do
      user_record = FactoryBot.create(:user_record)

      resolver = PermissionResolver.new(
        user_record:,
        space_record: nil,
        topic_record: nil
      )

      policy = resolver.resolve
      expect(policy).to be_a(SpaceGuestPolicy)
    end

    it "全てのパラメータがnilでもエラーにならないこと" do
      resolver = PermissionResolver.new(
        user_record: nil,
        space_record: nil,
        topic_record: nil
      )

      policy = resolver.resolve
      expect(policy).to be_a(SpaceGuestPolicy)
    end

    it "TopicがSpaceに属していなくてもエラーにならないこと" do
      user_record = FactoryBot.create(:user_record)
      space_record1 = FactoryBot.create(:space_record)
      space_record2 = FactoryBot.create(:space_record)
      topic_record = FactoryBot.create(:topic_record, space_record: space_record2)

      # space1のメンバー
      FactoryBot.create(:space_member_record,
        user_record:,
        space_record: space_record1,
        role: SpaceMemberRole::Member.serialize)

      # space1を指定し、space2のtopicを指定（不整合なデータ）
      resolver = PermissionResolver.new(
        user_record:,
        space_record: space_record1,
        topic_record:
      )

      # エラーにならず、Space Memberの権限が適用される
      policy = resolver.resolve
      expect(policy).to be_a(SpaceMemberPolicy)
    end

    it "非アクティブなSpaceMemberでも権限判定されること" do
      user_record = FactoryBot.create(:user_record)
      space_record = FactoryBot.create(:space_record)

      # 非アクティブなSpace Ownerを作成
      FactoryBot.create(:space_member_record,
        user_record:,
        space_record:,
        role: SpaceMemberRole::Owner.serialize,
        active: false)

      resolver = PermissionResolver.new(
        user_record:,
        space_record:,
        topic_record: nil
      )

      # 非アクティブでもOwnerPolicyが返される（active判定はPolicy側で行う）
      policy = resolver.resolve
      expect(policy).to be_a(SpaceOwnerPolicy)
    end
  end
end
