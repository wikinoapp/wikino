# typed: strict
# frozen_string_literal: true

module UserSessions
  class CreationForm < ApplicationForm
    attribute :email, :string
    attribute :password, :string

    validates :email, email: true, presence: true
    validates :password, presence: true
    validate :authentication

    sig { returns(T.nilable(UserRecord)) }
    def user_record
      @user ||= T.let(UserRecord.kept.find_by(email:), T.nilable(UserRecord))
    end

    sig { void }
    private def authentication
      unless user_record&.user_password_record&.authenticate(password)
        errors.add(:base, :unauthenticated)
      end
    end
  end
end