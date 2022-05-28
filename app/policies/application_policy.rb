# typed: strict
# frozen_string_literal: true

class ApplicationPolicy
  extend T::Sig

  sig { returns(ApplicationRecord) }
  attr_reader :user, :record

  sig { params(user: User, record: ApplicationRecord).void }
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

  class Scope
    extend T::Sig

    sig { params(user: User, scope: ActiveRecord::Relation).void }
    def initialize(user, scope)
      @user = user
      @scope = scope
    end

    sig { void }
    def resolve
      raise NotImplementedError, "You must define #resolve in #{self.class}"
    end

    private

    sig { returns(ApplicationRecord) }
    attr_reader :user

    sig { returns(ActiveRecord::Relation) }
    attr_reader :scope
  end
end
