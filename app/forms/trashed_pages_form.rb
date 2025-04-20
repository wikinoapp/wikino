# typed: strict
# frozen_string_literal: true

class TrashedPagesForm < ApplicationForm
  sig { returns(T.nilable(User)) }
  attr_accessor :user

  sig { returns(T.nilable(T::Array[T::Wikino::DatabaseId])) }
  attr_accessor :page_ids

  validates :user, presence: true
  validates :page_ids, presence: true
  validate :restoring_ability

  sig { returns(Page::PrivateRelation) }
  private def pages
    Page.where(id: page_ids)
  end

  sig { void }
  private def restoring_ability
    return if user.nil?

    unless user.not_nil!.joined_all_topics?(topic_ids: pages.pluck(:topic_id))
      errors.add(:base, :not_joined_topic_exists)
    end
  end
end
