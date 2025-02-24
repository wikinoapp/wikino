# typed: strict
# frozen_string_literal: true

class PageEntity < ApplicationEntity
  sig { returns(T::Wikino::DatabaseId) }
  attr_reader :database_id

  sig { returns(Integer) }
  attr_reader :number

  sig { returns(String) }
  attr_reader :title

  sig { returns(String) }
  attr_reader :body

  sig { returns(String) }
  attr_reader :body_html

  sig { returns(ActiveSupport::TimeWithZone) }
  attr_reader :modified_at

  sig { returns(ActiveSupport::TimeWithZone) }
  attr_reader :published_at

  sig { returns(T.nilable(ActiveSupport::TimeWithZone)) }
  attr_reader :pinned_at

  sig { returns(SpaceEntity) }
  attr_reader :space_entity

  sig { returns(TopicEntity) }
  attr_reader :topic_entity

  sig do
    params(
      database_id: T::Wikino::DatabaseId,
      number: Integer,
      title: String,
      body: String,
      body_html: String,
      pinned_at: T.nilable(ActiveSupport::TimeWithZone),
      modified_at: ActiveSupport::TimeWithZone,
      published_at: ActiveSupport::TimeWithZone,
      space_entity: SpaceEntity,
      topic_entity: TopicEntity
    ).void
  end
  def initialize(
    database_id:,
    number:,
    title:,
    body:,
    body_html:,
    pinned_at:,
    modified_at:,
    published_at:,
    space_entity:,
    topic_entity:
  )
    @database_id = database_id
    @number = number
    @title = title
    @body = body
    @body_html = body_html
    @pinned_at = pinned_at
    @modified_at = modified_at
    @published_at = published_at
    @space_entity = space_entity
    @topic_entity = topic_entity
  end

  sig { returns(T::Boolean) }
  def pinned?
    pinned_at.present?
  end
end
