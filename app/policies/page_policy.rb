# typed: strict
# frozen_string_literal: true

class PagePolicy < ApplicationPolicy
  include PolicyConcerns::SpaceContext

  sig { returns(T::Boolean) }
  def can_update?
    if same_space_member?
      return space_member_record!.active? &&
          space_member_record!.joined_topics.where(id: page_record.topic_id).exists?
    end

    false
  end

  sig { returns(PageRecord) }
  private def page_record
    T.cast(record, PageRecord)
  end
end
