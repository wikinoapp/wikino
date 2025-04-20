# typed: strict
# frozen_string_literal: true

class Page
  class PolicyRepository < ApplicationRepository
    sig { params(user: User, page: Page).returns(Page::Policy) }
    def build(user:, page:)
      user_record = UserRecord.find(user.database_id)
      page_record = PageRecord.find(page.database_id)

      Page::Policy.new(
        can_trash: user_record.can_trash_page?(page_record:)
      )
    end
  end
end
