# typed: strict
# frozen_string_literal: true

class TopicPolicy < ApplicationPolicy
  extend T::Sig

  sig { returns(T::Boolean) }
  def show?
    viewer.can_update_topic?(topic:)
  end

  sig { returns(T::Boolean) }
  def update?
    viewer.can_update_topic?(topic:)
  end

  sig { returns(T::Boolean) }
  def destroy?
    viewer.can_destroy_topic?(topic:)
  end

  sig { returns(Topic) }
  private def topic
    T.cast(record, Topic)
  end
end
