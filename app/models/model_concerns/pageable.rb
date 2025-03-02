# typed: strict
# frozen_string_literal: true

module ModelConcerns
  module Pageable
    include Kernel

    extend ActiveSupport::Concern
    extend T::Sig

    sig { returns(Page) }
    def original_page
      (instance_of?(DraftPage) ? T.bind(self, DraftPage).page : T.bind(self, Page)).not_nil!
    end

    T::Sig::WithoutRuntime.sig { returns(Page::PrivateRelation) }
    def linked_pages
      pages = space.not_nil!.pages.where(id: linked_page_ids)

      if Current.viewer!.joined_space?(space:)
        pages
      else
        pages.joins(:topic).merge(Topic.visibility_public)
      end
    end

    sig do
      params(
        space_viewer: ModelConcerns::SpaceViewable,
        before: T.nilable(String),
        after: T.nilable(String),
        link_limit: Integer,
        backlink_limit: Integer
      ).returns(LinkListEntity)
    end
    def fetch_link_list_entity(space_viewer:, before: nil, after: nil, link_limit: 15, backlink_limit: 14)
      added_page_ids = [id]

      cursor_paginate_page = linked_pages.where.not(id: added_page_ids).preload(:topic).cursor_paginate(
        after:,
        before:,
        limit: link_limit,
        order: {modified_at: :desc, id: :desc}
      ).fetch
      pages = cursor_paginate_page.records

      link_entities = fetch_link_entities(space_viewer:, pages:, added_page_ids:, backlink_limit:)
      pagination_entity = PaginationEntity.from_cursor_paginate(cursor_paginate_page:)

      LinkListEntity.new(
        link_entities:,
        pagination_entity:
      )
    end

    sig { params(editor: SpaceMember).void }
    def link!(editor:)
      location_keys = PageLocationKey.scan_text(text: body, current_topic: topic)
      topics = space.not_nil!.topics.where(name: location_keys.map(&:topic_name).uniq)

      linked_pages = location_keys.each_with_object([]) do |location_key, ary|
        page_topic = topics.find { |topic| topic.name == location_key.topic_name }

        if page_topic
          ary << editor.create_linked_page!(topic: page_topic, title: location_key.page_title)
        end
      end

      update!(linked_page_ids: linked_pages.pluck(:id))

      nil
    end

    sig do
      params(
        space_viewer: ModelConcerns::SpaceViewable,
        pages: Page::PrivateAssociationRelation,
        added_page_ids: T::Array[T::Wikino::DatabaseId],
        backlink_limit: Integer
      ).returns(T::Array[LinkEntity])
    end
    private def fetch_link_entities(space_viewer:, pages:, added_page_ids:, backlink_limit:)
      pages.map do |page|
        added_page_ids << page.id

        cursor_paginate_page = page.backlinked_pages.where.not(id: added_page_ids).preload(:topic).cursor_paginate(
          after: nil,
          before: nil,
          limit: backlink_limit,
          order: {modified_at: :desc, id: :desc}
        ).fetch
        backlinked_pages = cursor_paginate_page.records
        added_page_ids.concat(backlinked_pages.pluck(:id))

        backlink_entities = backlinked_pages.map do |backlinked_page|
          BacklinkEntity.new(page_entity: backlinked_page.to_entity(space_viewer:))
        end

        backlink_list_entity = BacklinkListEntity.new(
          backlink_entities:,
          pagination_entity: PaginationEntity.from_cursor_paginate(cursor_paginate_page:)
        )

        LinkEntity.new(
          page_entity: page.to_entity(space_viewer:),
          backlink_list_entity:
        )
      end
    end
  end
end
