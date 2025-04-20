# typed: strict
# frozen_string_literal: true

class Page
  class PolicyRepository < ApplicationRepository
    sig { params(user_record: UserRecord, page_record: PageRecord).returns(Page::Policy) }
    def build(user_record:, page_record:)
      Page::Policy.new(
        can_trash: user_record.can_trash_page?(page_record:)
      )
    end
  end
end
