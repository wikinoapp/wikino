# typed: strict
# frozen_string_literal: true

class TopicEntity < ApplicationEntity
  sig { returns(T::Wikino::DatabaseId) }
  attr_reader :database_id

  sig { returns(Integer) }
  attr_reader :number

  sig { returns(String) }
  attr_reader :name

  sig { returns(TopicVisibility) }
  attr_reader :visibility

  sig { returns(T::Boolean) }
  attr_reader :viewer_can_create_page
  alias_method :viewer_can_create_page?, :viewer_can_create_page

  sig do
    params(
      database_id: T::Wikino::DatabaseId,
      number: Integer,
      name: String,
      visibility: TopicVisibility,
      viewer_can_create_page: T::Boolean
    ).void
  end
  def initialize(database_id:, number:, name:, visibility:, viewer_can_create_page:)
    @database_id = database_id
    @number = number
    @name = name
    @visibility = visibility
    @viewer_can_create_page = viewer_can_create_page
  end

  sig { returns(T::Boolean) }
  def visibility_public?
    visibility == TopicVisibility::Public
  end
end
