# typed: strict
# frozen_string_literal: true

module ControllerConcerns
  # Topic関連のヘルパーメソッドを提供するconcern
  module TopicAware
    extend T::Sig
    extend ActiveSupport::Concern
    include SpaceAware

    # 現在のユーザーのTopicメンバーレコードを取得
    # @param topic_record [TopicRecord] 対象のTopicレコード
    # @return [TopicMemberRecord, nil] TopicMemberレコード（未参加の場合はnil）
    sig { params(topic_record: TopicRecord).returns(T.nilable(TopicMemberRecord)) }
    def current_topic_member_record(topic_record:)
      user_record = current_user_record
      return if user_record.nil?

      user_record.topic_member_records.find_by(topic_record:)
    end

    # 現在のユーザーのTopicメンバーレコードを取得（必須）
    # @param topic_record [TopicRecord] 対象のTopicレコード
    # @return [TopicMemberRecord] TopicMemberレコード
    # @raise [NoMethodError] TopicMemberRecordが存在しない場合
    sig { params(topic_record: TopicRecord).returns(TopicMemberRecord) }
    def current_topic_member_record!(topic_record:)
      current_topic_member_record(topic_record:).not_nil!
    end

    # リクエストパラメータからTopicレコードを取得
    # @return [TopicRecord, nil] Topicレコード
    sig { returns(T.nilable(TopicRecord)) }
    def current_topic_record
      return @current_topic_record if defined?(@current_topic_record)

      space_record = current_space_record
      @current_topic_record = T.let(
        if params[:topic_number].present? && space_record
          space_record.topic_records.find_by(number: params[:topic_number])
        end,
        T.nilable(TopicRecord)
      )
    end

    # リクエストパラメータからTopicレコードを取得（必須）
    # @return [TopicRecord] Topicレコード
    # @raise [NoMethodError] Topicレコードが存在しない場合
    sig { returns(TopicRecord) }
    def current_topic_record!
      current_topic_record.not_nil!
    end

    # Topic用のPolicyインスタンスを取得
    # @param topic_record [TopicRecord] 対象のTopicレコード
    # @return [ApplicationPolicy] 適切なPolicyインスタンス（Topic権限を考慮）
    sig { params(topic_record: TopicRecord).returns(ApplicationPolicy) }
    def topic_policy_for(topic_record:)
      space_record = topic_record.space_record.not_nil!
      space_member_record = current_space_member_record(space_record:)

      # Space Ownerの場合は、TopicでもOwner権限を持つ
      if space_member_record && space_member_record.role == SpaceMemberRole::Owner.serialize
        return SpaceOwnerPolicy.new(
          user_record: current_user_record.not_nil!,
          space_member_record:
        )
      end

      # Topic Adminの場合
      topic_member_record = current_topic_member_record(topic_record:)
      if topic_member_record && topic_member_record.role == TopicMemberRole::Admin.serialize
        return TopicAdminPolicy.new(
          user_record: current_user_record.not_nil!,
          topic_member_record:,
          space_member_record: space_member_record.not_nil!
        )
      end

      # その他の場合は通常のSpace Policyを返す
      space_policy_for(space_record:)
    end
  end
end
