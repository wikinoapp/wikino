# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe TopicMemberPolicy do
  it "スペースメンバーが編集提案を作成できること" do
    user_record = FactoryBot.create(:user_record)
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record:)
    space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:, active: true)
    topic_member_record = FactoryBot.create(:topic_member_record, space_member_record:, topic_record:, role: "member")

    policy = TopicMemberPolicy.new(user_record:, space_member_record:, topic_member_record:)

    expect(policy.can_create_edit_suggestion?).to be true
  end

  it "非アクティブなメンバーは編集提案を作成できないこと" do
    user_record = FactoryBot.create(:user_record)
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record:)
    space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:, active: false)
    topic_member_record = FactoryBot.create(:topic_member_record, space_member_record:, topic_record:, role: "member")

    policy = TopicMemberPolicy.new(user_record:, space_member_record:, topic_member_record:)

    expect(policy.can_create_edit_suggestion?).to be false
  end

  it "作成者は自分の編集提案を更新できること" do
    user_record = FactoryBot.create(:user_record)
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record:)
    space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:, active: true)
    topic_member_record = FactoryBot.create(:topic_member_record, space_member_record:, topic_record:, role: "member")
    edit_suggestion_record = FactoryBot.create(:edit_suggestion_record,
      space: space_record,
      topic: topic_record,
      created_user: user_record)

    policy = TopicMemberPolicy.new(user_record:, space_member_record:, topic_member_record:)

    expect(policy.can_update_edit_suggestion?(edit_suggestion_record:)).to be true
  end

  it "他のユーザーは編集提案を更新できないこと" do
    user_record = FactoryBot.create(:user_record)
    other_user_record = FactoryBot.create(:user_record)
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record:)
    space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:, active: true)
    topic_member_record = FactoryBot.create(:topic_member_record, space_member_record:, topic_record:, role: "member")
    edit_suggestion_record = FactoryBot.create(:edit_suggestion_record,
      space: space_record,
      topic: topic_record,
      created_user: other_user_record)

    policy = TopicMemberPolicy.new(user_record:, space_member_record:, topic_member_record:)

    expect(policy.can_update_edit_suggestion?(edit_suggestion_record:)).to be false
  end

  it "トピック管理者は編集提案を反映できること" do
    user_record = FactoryBot.create(:user_record)
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record:)
    space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:, active: true)
    topic_member_record = FactoryBot.create(:topic_member_record, space_member_record:, topic_record:, role: "admin")
    edit_suggestion_record = FactoryBot.create(:edit_suggestion_record,
      space: space_record,
      topic: topic_record,
      created_user: user_record)

    policy = TopicMemberPolicy.new(user_record:, space_member_record:, topic_member_record:)

    expect(policy.can_apply_edit_suggestion?(edit_suggestion_record:)).to be true
  end

  it "一般メンバーも編集提案を反映できること" do
    user_record = FactoryBot.create(:user_record)
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record:)
    space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:, active: true)
    topic_member_record = FactoryBot.create(:topic_member_record, space_member_record:, topic_record:, role: "member")
    edit_suggestion_record = FactoryBot.create(:edit_suggestion_record,
      space: space_record,
      topic: topic_record,
      created_user: user_record)

    policy = TopicMemberPolicy.new(user_record:, space_member_record:, topic_member_record:)

    expect(policy.can_apply_edit_suggestion?(edit_suggestion_record:)).to be true
  end

  it "作成者、トピック管理者、トピックメンバーは編集提案をクローズできること" do
    user_record = FactoryBot.create(:user_record)
    admin_record = FactoryBot.create(:user_record)
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record:)

    # 作成者
    space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:, active: true)
    topic_member_record = FactoryBot.create(:topic_member_record, space_member_record:, topic_record:, role: "member")
    edit_suggestion_record = FactoryBot.create(:edit_suggestion_record,
      space: space_record,
      topic: topic_record,
      created_user: user_record)

    creator_policy = TopicMemberPolicy.new(user_record:, space_member_record:, topic_member_record:)
    expect(creator_policy.can_close_edit_suggestion?(edit_suggestion_record:)).to be true

    # 管理者
    admin_space_member_record = FactoryBot.create(:space_member_record, user_record: admin_record, space_record:, active: true)
    admin_topic_member_record = FactoryBot.create(:topic_member_record, space_member_record: admin_space_member_record, topic_record:, role: "admin")

    admin_policy = TopicMemberPolicy.new(user_record: admin_record, space_member_record: admin_space_member_record, topic_member_record: admin_topic_member_record)
    expect(admin_policy.can_close_edit_suggestion?(edit_suggestion_record:)).to be true
  end

  it "トピックメンバーは編集提案にコメントできること" do
    user_record = FactoryBot.create(:user_record)
    space_record = FactoryBot.create(:space_record)
    topic_record = FactoryBot.create(:topic_record, space_record:)
    space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:, active: true)
    topic_member_record = FactoryBot.create(:topic_member_record, space_member_record:, topic_record:, role: "member")
    edit_suggestion_record = FactoryBot.create(:edit_suggestion_record,
      space: space_record,
      topic: topic_record,
      created_user: user_record)

    policy = TopicMemberPolicy.new(user_record:, space_member_record:, topic_member_record:)

    expect(policy.can_comment_on_edit_suggestion?(edit_suggestion_record:)).to be true
  end
end
