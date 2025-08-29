# typed: strict
# frozen_string_literal: true

# 全てのPolicyクラスの基底クラス
class ApplicationPolicy
  extend T::Sig
  extend T::Helpers
  abstract!

  sig { params(user_record: T.nilable(UserRecord)).void }
  def initialize(user_record:)
    @user_record = user_record
  end

  private

  sig { returns(T.nilable(UserRecord)) }
  attr_reader :user_record
end
