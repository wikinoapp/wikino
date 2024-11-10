# typed: strict
# frozen_string_literal: true

class ApplicationPolicy
  extend T::Sig

  sig { returns(T.nilable(User)) }
  attr_reader :user

  sig { returns(ApplicationRecord) }
  attr_reader :record

  sig { params(user: T.nilable(User), record: ApplicationRecord).void }
  def initialize(user, record)
    @user = user
    @record = record
  end

  sig { returns(T::Boolean) }
  def index?
    false
  end

  sig { returns(T::Boolean) }
  def show?
    false
  end

  sig { returns(T::Boolean) }
  def create?
    false
  end

  sig { returns(T::Boolean) }
  def new?
    create?
  end

  sig { returns(T::Boolean) }
  def update?
    false
  end

  sig { returns(T::Boolean) }
  def edit?
    update?
  end

  sig { returns(T::Boolean) }
  def destroy?
    false
  end
end
