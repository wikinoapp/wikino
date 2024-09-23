# typed: strict
# frozen_string_literal: true

class ApplicationPolicy
  extend T::Sig

  sig { returns(User) }
  attr_reader :viewer

  sig { returns(ApplicationRecord) }
  attr_reader :record

  sig { params(viewer: User, record: ApplicationRecord).void }
  def initialize(viewer, record)
    @viewer = viewer
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
    extend T::Helpers

    abstract!

    sig { params(viewer: User, scope: T.any(ActiveRecord::Relation, T.class_of(ApplicationRecord))).void }
    def initialize(viewer, scope)
      @viewer = viewer
      @scope = scope
    end

    sig { abstract.returns(ActiveRecord::Relation) }
    def resolve
    end

    sig { returns(User) }
    attr_reader :viewer
    private :viewer

    sig { returns(T.any(ActiveRecord::Relation, T.class_of(ApplicationRecord))) }
    attr_reader :scope
    private :scope
  end
end
