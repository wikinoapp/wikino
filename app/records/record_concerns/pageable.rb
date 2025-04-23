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

    T::Sig::WithoutRuntime.sig { returns(PageRecord::PrivateRelation) }
    def linked_pages
      pages = space_record.not_nil!.page_records.where(id: linked_page_ids)

      if Current.viewer!.joined_space?(space: space_record)
        pages
      else
        pages.joins(:topic_record).merge(TopicRecord.visibility_public)
      end
    end

    sig { params(editor: SpaceMemberRecord).void }
    def link!(editor:)
      location_keys = PageLocationKey.scan_text(text: body, current_topic: topic_record)
      topics = space_record.not_nil!.topic_records.where(name: location_keys.map(&:topic_name).uniq)

      linked_pages = location_keys.each_with_object([]) do |location_key, ary|
        page_topic = topics.find { |topic| topic.name == location_key.topic_name }

        if page_topic
          ary << editor.create_linked_page!(topic_record: page_topic, title: location_key.page_title)
        end
      end

      update!(linked_page_ids: linked_pages.pluck(:id))

      nil
    end
  end
end
