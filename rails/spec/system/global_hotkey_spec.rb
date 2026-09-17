# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe "Global Hotkey", type: :system do
  it "検索ページ以外でsキーまたは/キーを押すと検索ページに遷移すること", :js do
    user_record = create(:user_record, :with_password)
    sign_in(user_record:)

    visit_with_global_hotkey("/settings")

    # sキーを押すと検索ページに遷移
    page.driver.browser.action.send_keys("s").perform
    expect(page).to have_current_path(search_path)

    # 設定ページに戻る
    visit_with_global_hotkey("/settings")

    # /キーを押すと検索ページに遷移
    page.driver.browser.action.send_keys("/").perform
    expect(page).to have_current_path(search_path)
  end

  # Visits the given path and waits until the global-hotkey controller is ready.
  #
  # [Ja] 指定パスへ遷移し、global-hotkey コントローラの connect 完了まで待つ。
  private def visit_with_global_hotkey(path)
    visit path
    wait_for_global_hotkey_ready
  end

  # Waits until the global-hotkey Stimulus controller has connected on the
  # current page. @github/hotkey registers its keydown listener inside the
  # controller's connect(), so sending a key before connect() runs drops the
  # event and leaves the page on its current path -- the cause of this spec's
  # flakiness. Once getControllerForElementAndIdentifier returns the instance,
  # connect() (and therefore install()) has already run synchronously.
  #
  # [Ja] 現在のページで global-hotkey Stimulus コントローラの connect 完了を待つ。
  # @github/hotkey はコントローラの connect() 内で keydown リスナを登録するため、
  # connect() の実行前にキーを送るとイベントが失われ、ページが現在のパスに留まる
  # (この spec の flaky の原因)。getControllerForElementAndIdentifier がインスタンスを
  # 返した時点で connect() (= install()) は同期的に実行済みである。
  private def wait_for_global_hotkey_ready
    deadline = Process.clock_gettime(Process::CLOCK_MONOTONIC) + Capybara.default_max_wait_time

    loop do
      return if global_hotkey_connected?

      if Process.clock_gettime(Process::CLOCK_MONOTONIC) > deadline
        raise "global-hotkey controller was not connected within #{Capybara.default_max_wait_time}s"
      end

      sleep 0.05
    end
  end

  # Returns true once the global-hotkey controller is connected to its element.
  #
  # [Ja] global-hotkey コントローラが要素に connect 済みなら true を返す。
  private def global_hotkey_connected?
    page.evaluate_script(<<~JS)
      (() => {
        const element = document.querySelector('[data-controller~="global-hotkey"]');
        return Boolean(
          element &&
          window.Stimulus &&
          window.Stimulus.getControllerForElementAndIdentifier(element, "global-hotkey")
        );
      })()
    JS
  end
end
