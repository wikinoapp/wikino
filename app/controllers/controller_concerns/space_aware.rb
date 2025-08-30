# typed: strict
# frozen_string_literal: true

module ControllerConcerns
  # Space関連のヘルパーメソッドを提供するconcern
  module SpaceAware
    extend T::Sig
    extend ActiveSupport::Concern

    included do
      # Authenticatableが必要
      unless included_modules.include?(ControllerConcerns::Authenticatable)
        raise "SpaceAware requires Authenticatable to be included first"
      end
    end

    # 現在のユーザーのSpaceメンバーレコードを取得
    # @param space_record [SpaceRecord] 対象のSpaceレコード
    # @return [SpaceMemberRecord, nil] SpaceMemberレコード（未参加の場合はnil）
    sig { params(space_record: SpaceRecord).returns(T.nilable(SpaceMemberRecord)) }
    def current_space_member_record(space_record:)
      user_record = current_user_record
      return if user_record.nil?

      user_record.space_member_record(space_record:)
    end

    # 現在のユーザーのSpaceメンバーレコードを取得（必須）
    # @param space_record [SpaceRecord] 対象のSpaceレコード
    # @return [SpaceMemberRecord] SpaceMemberレコード
    # @raise [NoMethodError] SpaceMemberRecordが存在しない場合
    sig { params(space_record: SpaceRecord).returns(SpaceMemberRecord) }
    def current_space_member_record!(space_record:)
      current_user_record!.space_member_record(space_record:).not_nil!
    end

    # Space用のPolicyインスタンスを取得
    # @param space_record [SpaceRecord] 対象のSpaceレコード
    # @return [T::Wikino::SpacePolicyInstance] 適切なPolicyインスタンス
    sig { params(space_record: SpaceRecord).returns(T::Wikino::SpacePolicyInstance) }
    def space_policy_for(space_record:)
      space_member_record = current_space_member_record(space_record:)
      SpacePolicyFactory.build(
        user_record: current_user_record,
        space_member_record:
      )
    end

    # リクエストパラメータからSpaceレコードを取得
    # @return [SpaceRecord, nil] Spaceレコード
    sig { returns(T.nilable(SpaceRecord)) }
    def current_space_record
      return @current_space_record if defined?(@current_space_record)

      @current_space_record = T.let(
        if params[:space_identifier].present?
          SpaceRecord.find_by(identifier: params[:space_identifier])
        end,
        T.nilable(SpaceRecord)
      )
    end

    # リクエストパラメータからSpaceレコードを取得（必須）
    # @return [SpaceRecord] Spaceレコード
    # @raise [NoMethodError] Spaceレコードが存在しない場合
    sig { returns(SpaceRecord) }
    def current_space_record!
      current_space_record.not_nil!
    end
  end
end
