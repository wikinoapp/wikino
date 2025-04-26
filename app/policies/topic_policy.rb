# typed: strict
# frozen_string_literal: true

class TopicPolicy < ApplicationPolicy
  include PolicyConcerns::SpaceContext

  sig { returns(T::Boolean) }
  def create_page?
    space_member_record&.topic_records&.where(id: topic_record.id)&.exists? == true
  end

  sig { returns(TopicRecord) }
  private def topic_record
    T.cast(record, TopicRecord)
  end
end
