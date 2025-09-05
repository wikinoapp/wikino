# typed: strict
# frozen_string_literal: true

# ロールに応じた適切なTopic Policyクラスを生成するFactory
class TopicPolicyFactory
  extend T::Sig

  sig do
    params(
      user_record: T.nilable(UserRecord),
      space_member_record: T.nilable(SpaceMemberRecord),
      topic_member_record: T.nilable(TopicMemberRecord)
    ).returns(Types::TopicPolicyInstance)
  end
  def self.build(user_record:, space_member_record: nil, topic_member_record: nil)
    # 非メンバーの場合（スペースメンバーでもトピックメンバーでもない）
    if space_member_record.nil? || topic_member_record.nil?
      return TopicGuestPolicy.new(user_record:)
    end

    # space_member_recordとtopic_member_recordが存在する場合、user_recordも必ず存在するはず
    if user_record.nil?
      raise ArgumentError, "user_record must not be nil when space_member_record and topic_member_record are present"
    end

    # Space Ownerの場合は常にTopicOwnerPolicy
    if space_member_record.role == SpaceMemberRole::Owner.serialize
      return TopicOwnerPolicy.new(user_record:, space_member_record:)
    end

    # トピックメンバーのロールに応じたPolicyを返す
    case topic_member_record.role
    when TopicMemberRole::Admin.serialize
      TopicAdminPolicy.new(user_record:, space_member_record:, topic_member_record:)
    when TopicMemberRole::Member.serialize
      TopicMemberPolicy.new(user_record:, space_member_record:, topic_member_record:)
    else
      raise ArgumentError, "Unknown topic member role: #{topic_member_record.role}"
    end
  end
end
