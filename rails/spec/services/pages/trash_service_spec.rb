# typed: false
# frozen_string_literal: true

RSpec.describe Pages::TrashService, type: :service do
  describe "#call" do
    it "ページをゴミ箱に移動できること" do
      page_record = create(:page_record)
      updated_at = page_record.updated_at

      expect(page_record.trashed_at).to be_nil

      result = Pages::TrashService.new.call(page_record:)

      expect(result.page_record).to eq(page_record)
      expect(result.page_record.trashed_at).to be_present
      expect(result.page_record.updated_at).to be > updated_at
    end
  end
end
