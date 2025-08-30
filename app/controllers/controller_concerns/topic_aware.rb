# typed: strict
# frozen_string_literal: true

module ControllerConcerns
  # Topic関連のヘルパーメソッドを提供するconcern
  module TopicAware
    extend T::Sig
    extend ActiveSupport::Concern

    include SpaceAware

    # 現在のユーザーのTopicメンバーレコードを取得
    sig { params(topic_record: TopicRecord).returns(T.nilable(TopicMemberRecord)) }
    def current_topic_member_record(topic_record:)
      current_user_record&.topic_member_records&.find_by(topic_record:)
    end

    # 現在のユーザーのTopicメンバーレコードを取得（必須）
    sig { params(topic_record: TopicRecord).returns(TopicMemberRecord) }
    def current_topic_member_record!(topic_record:)
      current_topic_member_record(topic_record:).not_nil!
    end

    # リクエストパラメータからTopicレコードを取得
    sig { returns(T.nilable(TopicRecord)) }
    def current_topic_record
      return @current_topic_record if defined?(@current_topic_record)

      @current_topic_record = T.let(
        if params[:topic_number].present? && params[:space_identifier].present?
          space_record = SpaceRecord.kept.find_by(identifier: params[:space_identifier])
          space_record&.topic_records&.find_by(number: params[:topic_number])
        end,
        T.nilable(TopicRecord)
      )
    end

    # リクエストパラメータからTopicレコードを取得（必須）
    sig { returns(TopicRecord) }
    def current_topic_record!
      current_topic_record.not_nil!
    end

    # Topic用のPolicyインスタンスを取得
    # SpacePolicyInstanceのtopic_policy_forメソッドを呼び出す
    sig { params(topic_record: TopicRecord).returns(T::Wikino::TopicPolicyInstance) }
    def topic_policy_for(topic_record:)
      space_policy = space_policy_for(space_record: topic_record.space_record.not_nil!)

      # SpacePolicyInstanceのtopic_policy_forメソッドを呼び出す
      space_policy.topic_policy_for(topic_record:)
    end
  end
end
