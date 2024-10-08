# typed: strict
# frozen_string_literal: true

class EditPageForm < ApplicationForm
  sig { returns(T.nilable(User)) }
  attr_accessor :viewer

  attribute :topic_number, :integer
  attribute :title, :string
  attribute :body, :string, default: ""

  validates :topic, presence: true
  validates :title, presence: true
  validates :body, presence: true, allow_blank: true

  sig { returns(T.nilable(Topic)) }
  def topic
    viewer&.viewable_topics&.find_by(number: topic_number)
  end

  sig { returns(Topic::PrivateRelation) }
  def viewable_topics
    viewer.not_nil!.viewable_topics
  end

  sig { returns(T::Boolean) }
  def autofocus_title?
    title.blank?
  end

  sig { returns(T::Boolean) }
  def autofocus_body?
    !autofocus_title?
  end
end
