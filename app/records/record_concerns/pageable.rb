# typed: strict
# frozen_string_literal: true

module RecordConcerns
  module Pageable
    include Kernel

    extend ActiveSupport::Concern
    extend T::Sig

    sig { returns(PageRecord) }
    def original_page
      (instance_of?(DraftPageRecord) ? T.bind(self, DraftPageRecord).page_record : T.bind(self, PageRecord)).not_nil!
    end

    sig { params(user_record: T.nilable(UserRecord)).returns(PageRecord::PrivateRelation) }
    def linked_pages(user_record:)
      page_records = space_record.not_nil!.page_records.available.where(id: linked_page_ids)

      if user_record&.joined_space?(space_record:)
        page_records
      else
        page_records.topics_visibility_public
      end
    end

    sig { params(editor_record: SpaceMemberRecord).void }
    def link!(editor_record:)
      location_keys = PageLocationKey.scan_text(text: body, current_topic: topic_record)
      topics = space_record.not_nil!.topic_records.where(name: location_keys.map(&:topic_name).uniq)

      linked_pages = location_keys.each_with_object([]) do |location_key, ary|
        page_topic = topics.find { |topic| topic.name == location_key.topic_name }

        if page_topic
          ary << editor_record.create_linked_page!(topic_record: page_topic, title: location_key.page_title)
        end
      end

      update!(linked_page_ids: linked_pages.pluck(:id))

      nil
    end
  end
end
